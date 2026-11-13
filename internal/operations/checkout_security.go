package operations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"leguiburger/internal/buyeraccess"
	"leguiburger/internal/requestguard"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

var errVerificationUnavailable = errors.New("verification provider unavailable")

type checkoutSecurity struct {
	db                                 *gorm.DB
	siteKey, secret, hostname, hashKey string
	client                             *http.Client
}
type riskState struct {
	SubjectHash                          string
	WindowStart                          time.Time
	ServerTime                           time.Time
	NewSessions, InvalidRequests, Orders int
	BlockedUntil                         *time.Time
	ReleasedUntil                        *time.Time
}

func newCheckoutSecurity(db *gorm.DB) *checkoutSecurity {
	origin, _ := url.Parse(os.Getenv("PUBLIC_SITE_URL"))
	hostname := ""
	if origin != nil {
		hostname = origin.Hostname()
	}
	key := os.Getenv("ABUSE_HASH_KEY")
	if key == "" {
		key = os.Getenv("JWT_SECRET")
	}
	return &checkoutSecurity{db: db, siteKey: os.Getenv("TURNSTILE_SITE_KEY"), secret: os.Getenv("TURNSTILE_SECRET_KEY"), hostname: hostname, hashKey: key, client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (s *checkoutSecurity) configured() bool {
	if s.siteKey == "" || s.secret == "" || s.hostname == "" {
		return false
	}
	if os.Getenv("APP_ENV") == "production" {
		for _, prefix := range []string{"1x0000", "2x0000", "3x0000", "0x0000"} {
			if strings.HasPrefix(s.siteKey, prefix) || strings.HasPrefix(s.secret, prefix) {
				return false
			}
		}
	}
	return true
}
func (s *checkoutSecurity) hash(value string) string {
	m := hmac.New(sha256.New, []byte(s.hashKey))
	m.Write([]byte(value))
	return hex.EncodeToString(m.Sum(nil))
}
func (s *checkoutSecurity) subject(r *http.Request) string {
	return s.hash("ip:" + requestguard.ClientIP(r))
}
func (s *checkoutSecurity) session(raw string) string { return s.hash("session:" + raw) }
func maskedIP(r *http.Request) string {
	ip, e := netip.ParseAddr(requestguard.ClientIP(r))
	if e != nil {
		return "Origen desconocido"
	}
	bits := 48
	if ip.Is4() {
		bits = 24
	}
	return netip.PrefixFrom(ip, bits).Masked().String()
}
func (st riskState) now() time.Time {
	if !st.ServerTime.IsZero() {
		return st.ServerTime
	}
	return time.Now()
}
func riskReason(st riskState) string {
	if st.now().Sub(st.WindowStart) > 15*time.Minute {
		return ""
	}
	if st.InvalidRequests >= 8 {
		return "invalid_requests"
	}
	if st.NewSessions >= 12 {
		return "session_churn"
	}
	if st.Orders >= 8 {
		return "repeated_orders"
	}
	return ""
}
func (s *checkoutSecurity) state(ctx context.Context, r *http.Request) (riskState, error) {
	var st riskState
	err := s.db.WithContext(ctx).Raw(`SELECT subject_hash,window_start,new_sessions,invalid_requests,orders,blocked_until,released_until,clock_timestamp() server_time FROM legui_private.abuse_subjects WHERE subject_hash=?`, s.subject(r)).Scan(&st).Error
	return st, err
}
func blockedRisk(st riskState) error {
	if st.BlockedUntil != nil && st.BlockedUntil.After(st.now()) {
		return &apiError{429, "CHECKOUT_BLOCKED", "Detectamos actividad inusual. Los intentos de compra están pausados temporalmente. Podés seguir consultando la carta y el estado de tu pedido."}
	}
	return nil
}
func (s *checkoutSecurity) blockCheck(ctx context.Context, r *http.Request) error {
	st, e := s.state(ctx, r)
	if e != nil {
		return e
	}
	return blockedRisk(st)
}
func (s *checkoutSecurity) observe(ctx context.Context, r *http.Request, tenant, signal, reason string) error {
	field := ""
	switch signal {
	case "session":
		field = "new_sessions"
	case "invalid":
		field = "invalid_requests"
	case "order":
		field = "orders"
	case "burst":
		field = "invalid_requests"
	default:
		return errors.New("invalid risk signal")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var st riskState
		// Only server-selected counters enter this SQL. Client content is never evidence.
		q := `INSERT INTO legui_private.abuse_subjects(subject_hash,label,` + field + `) VALUES(?,?,0) ON CONFLICT(subject_hash) DO UPDATE SET new_sessions=CASE WHEN abuse_subjects.window_start<clock_timestamp()-interval '15 minutes' THEN 0 ELSE abuse_subjects.new_sessions END,invalid_requests=CASE WHEN abuse_subjects.window_start<clock_timestamp()-interval '15 minutes' THEN 0 ELSE abuse_subjects.invalid_requests END,orders=CASE WHEN abuse_subjects.window_start<clock_timestamp()-interval '15 minutes' THEN 0 ELSE abuse_subjects.orders END,window_start=CASE WHEN abuse_subjects.window_start<clock_timestamp()-interval '15 minutes' THEN clock_timestamp() ELSE abuse_subjects.window_start END,last_seen_at=clock_timestamp() RETURNING subject_hash,window_start,new_sessions,invalid_requests,orders,blocked_until,clock_timestamp() server_time`
		if e := tx.Raw(q, s.subject(r), maskedIP(r)).Scan(&st).Error; e != nil {
			return e
		}
		// The same increment applies to new and existing windows.
		if e := tx.Exec(`UPDATE legui_private.abuse_subjects SET `+field+`=`+field+`+1 WHERE subject_hash=?`, st.SubjectHash).Error; e != nil {
			return e
		}
		if e := tx.Raw(`SELECT subject_hash,window_start,new_sessions,invalid_requests,orders,blocked_until,released_until,clock_timestamp() server_time FROM legui_private.abuse_subjects WHERE subject_hash=?`, st.SubjectHash).Scan(&st).Error; e != nil {
			return e
		}
		if signal == "burst" || st.InvalidRequests >= 30 {
			if e := tx.Exec(`UPDATE legui_private.abuse_subjects SET blocked_until=CASE WHEN blocked_until>clock_timestamp() THEN blocked_until ELSE clock_timestamp()+interval '15 minutes' END,released_until=NULL WHERE subject_hash=?`, st.SubjectHash).Error; e != nil {
				return e
			}
		}
		if reason == "" {
			reason = riskReason(st)
		}
		if reason == "" {
			return nil
		}
		facts, _ := json.Marshal(map[string]int{"new_sessions": st.NewSessions, "invalid_requests": st.InvalidRequests, "orders": st.Orders})
		return tx.Exec(`INSERT INTO legui_private.abuse_incidents(subject_hash,tenant_id,reason,window_start,evidence) VALUES(?,(SELECT id FROM public.tenants WHERE id=nullif(?,'')::uuid),?,?,?::jsonb) ON CONFLICT(subject_hash,reason,window_start) DO UPDATE SET occurrences=abuse_incidents.occurrences+1,evidence=excluded.evidence,last_seen_at=clock_timestamp(),resolved_at=NULL,resolution_note=NULL`, st.SubjectHash, tenant, reason, st.WindowStart, string(facts)).Error
	})
}
func (s *checkoutSecurity) requireOrderCheck(r *http.Request, tenant, raw, key string) (bool, error) {
	st, e := s.state(r.Context(), r)
	if e != nil {
		return false, e
	}
	if e = blockedRisk(st); e != nil {
		return false, e
	}
	if riskReason(st) == "" {
		return false, nil
	}
	var allowed bool
	e = s.db.WithContext(r.Context()).Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.abuse_clearances WHERE subject_hash=? AND session_hash=? AND tenant_id=? AND expires_at>clock_timestamp() AND (used_key IS NULL OR used_key=?))`, s.subject(r), s.session(raw), tenant, key).Scan(&allowed).Error
	if e != nil {
		return false, e
	}
	if !allowed {
		return false, &apiError{403, "HUMAN_VERIFICATION_REQUIRED", "Necesitamos una verificación adicional antes de confirmar tu compra."}
	}
	return true, nil
}
func (s *checkoutSecurity) consume(tx *gorm.DB, r *http.Request, tenant, raw, key string) error {
	q := tx.Exec(`UPDATE legui_private.abuse_clearances SET used_key=? WHERE subject_hash=? AND session_hash=? AND tenant_id=? AND expires_at>clock_timestamp() AND (used_key IS NULL OR used_key=?)`, key, s.subject(r), s.session(raw), tenant, key)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected != 1 {
		return &apiError{403, "HUMAN_VERIFICATION_REQUIRED", "La verificación venció. Completala nuevamente para confirmar."}
	}
	return nil
}
func (h *Handler) humanContext(r *http.Request) (string, string, error) {
	tenant, e := resourceID(r)
	if e != nil {
		return "", "", notFound()
	}
	if !buyeraccess.SameOrigin(r) {
		return "", "", forbidden()
	}
	session, ok := buyeraccess.CookieSession(r, tenant)
	if !ok {
		return "", "", forbidden()
	}
	// A closed/unpublished or foreign location never becomes a verification target.
	if _, e := buyerLocation(h.repo.DB.WithContext(r.Context()), tenant); e != nil {
		return "", "", e
	}
	if e := h.security.blockCheck(r.Context(), r); e != nil {
		return "", "", e
	}
	return tenant, session.Secret, nil
}
func (h *Handler) humanCheck(r *http.Request) (Result, error) {
	tenant, raw, e := h.humanContext(r)
	if e != nil {
		return Result{}, e
	}
	s := h.security
	cdata := s.hash("checkout:" + s.subject(r) + ":" + s.session(raw) + ":" + tenant)
	if r.Method == http.MethodGet {
		data, _ := json.Marshal(map[string]any{"configured": s.configured(), "site_key": s.siteKey, "action": "checkout", "cdata": cdata})
		return Result{Data: data}, nil
	}
	if !s.configured() {
		return Result{}, &apiError{503, "HUMAN_VERIFICATION_UNAVAILABLE", "La verificación no está disponible ahora. Esperá unos minutos y volvé a intentar."}
	}
	if e := buyeraccess.Rate(r.Context(), h.repo.DB, r, "human-check", 30); e != nil {
		return Result{}, e
	}
	in, e := decode[struct {
		Token string `json:"token"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	if len(in.Token) == 0 || len(in.Token) > 2048 {
		return Result{}, badInput("Completá la verificación")
	}
	if e = s.verify(r.Context(), in.Token, cdata); e != nil {
		if errors.Is(e, errVerificationUnavailable) {
			return Result{}, &apiError{503, "HUMAN_VERIFICATION_UNAVAILABLE", "La verificación está demorada. Volvé a intentar en unos minutos."}
		}
		_ = s.observe(r.Context(), r, tenant, "invalid", "captcha_failed")
		return Result{}, &apiError{403, "HUMAN_VERIFICATION_FAILED", "No pudimos confirmar la verificación. Volvé a completarla."}
	}
	e = s.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		token := tx.Exec(`INSERT INTO legui_private.abuse_verified_tokens(token_digest,expires_at) VALUES(?,clock_timestamp()+interval '5 minutes') ON CONFLICT DO NOTHING`, buyeraccess.Digest(in.Token))
		if token.Error != nil {
			return token.Error
		}
		if token.RowsAffected != 1 {
			return conflict("Esta verificación ya fue utilizada. Completá una nueva")
		}
		// Clearance is bound to IP, browser capability and location, and allows one order.
		return tx.Exec(`INSERT INTO legui_private.abuse_clearances(subject_hash,session_hash,tenant_id,expires_at) VALUES(?,?,?,clock_timestamp()+interval '5 minutes') ON CONFLICT(subject_hash,session_hash,tenant_id) DO UPDATE SET expires_at=excluded.expires_at,used_key=NULL`, s.subject(r), s.session(raw), tenant).Error
	})
	return Result{Data: json.RawMessage(`{"verified":true}`)}, e
}
func (s *checkoutSecurity) verify(ctx context.Context, token, cdata string) error {
	body := url.Values{"secret": {s.secret}, "response": {token}}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(body.Encode()))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, e := s.client.Do(req)
	if e != nil {
		return errVerificationUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errVerificationUnavailable
	}
	var out struct {
		Success    bool      `json:"success"`
		Hostname   string    `json:"hostname"`
		Action     string    `json:"action"`
		CData      string    `json:"cdata"`
		Time       time.Time `json:"challenge_ts"`
		ErrorCodes []string  `json:"error-codes"`
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 16384)).Decode(&out); e != nil {
		return errVerificationUnavailable
	}
	for _, code := range out.ErrorCodes {
		if isOneOf(code, "internal-error", "invalid-input-secret", "missing-input-secret") {
			return errVerificationUnavailable
		}
	}
	age := time.Since(out.Time)
	if !out.Success || out.Hostname != s.hostname || out.Action != "checkout" || out.CData != cdata || age < -30*time.Second || age > 5*time.Minute {
		return errors.New("invalid verification")
	}
	return nil
}
