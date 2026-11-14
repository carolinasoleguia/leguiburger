package operations

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"leguiburger/internal/buyeraccess"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func commerceSite() string {
	raw := strings.TrimRight(os.Getenv("PUBLIC_SITE_URL"), "/")
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(os.Getenv("APP_ENV") != "production" && u.Scheme == "http")) {
		return ""
	}
	return raw
}

const campaignAudience = `SELECT p.id,p.email FROM legui_private.shoppers p JOIN legui_private.shopper_preferences pref ON pref.shopper_id=p.id AND pref.brand_id=? AND pref.newsletter LEFT JOIN LATERAL(SELECT count(*) n,max(o.created_at) last_paid FROM legui_private.shopper_orders so JOIN public.orders o ON o.id=so.order_id JOIN public.tenants t ON t.id=o.tenant_id WHERE so.shopper_id=p.id AND t.brand_id=pref.brand_id AND o.payment_environment='production' AND o.total_paid>0 AND o.order_status_id NOT IN(SELECT id FROM public.order_statuses_lookup WHERE code IN ('canceled','rejected'))) stats ON true WHERE p.active AND p.email_verified AND(?='all' OR(pref.personalization AND((?='frequent' AND stats.n>=3) OR(?='recover' AND stats.last_paid<clock_timestamp()-interval '30 days') OR(?='new' AND stats.n<=1))))`

func audience(tx *gorm.DB, brand, segment string) ([]struct{ ID, Email string }, error) {
	var rows []struct{ ID, Email string }
	e := tx.Raw(campaignAudience+" ORDER BY p.id LIMIT 5001", brand, segment, segment, segment, segment).Scan(&rows).Error
	return rows, e
}
func campaignData(tx *gorm.DB, brand, id string) (json.RawMessage, error) {
	return jsonQuery(tx, `SELECT (to_jsonb(c)-'body_cipher'||jsonb_build_object('recipients',(SELECT count(*) FROM legui_private.newsletter_recipients WHERE campaign_id=c.id),'delivered',(SELECT count(*) FROM legui_private.mail_outbox WHERE campaign_id=c.id AND delivered_at IS NOT NULL),'failed',(SELECT count(*) FROM legui_private.mail_outbox WHERE campaign_id=c.id AND delivered_at IS NULL AND attempts>=8),'pending',(SELECT count(*) FROM legui_private.newsletter_recipients r LEFT JOIN legui_private.mail_outbox m ON m.id=r.outbox_id WHERE r.campaign_id=c.id AND(r.status='pending' OR(m.body_cipher IS NOT NULL AND m.delivered_at IS NULL AND m.attempts<8)))))::text payload FROM legui_private.newsletter_campaigns c WHERE c.id=? AND c.brand_id=?`, id, brand)
}
func (h *Handler) campaigns(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	id := r.PathValue("id")
	configured := auth.MailConfigured() && commerceSite() != ""
	if r.Method == "GET" {
		if id == "" {
			return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('mail_configured',?::boolean,'items',coalesce(jsonb_agg(to_jsonb(v) ORDER BY v.created_at DESC),'[]'::jsonb))::text payload FROM(SELECT c.id,c.subject,c.segment,c.state,c.created_at,(SELECT count(*) FROM legui_private.newsletter_recipients WHERE campaign_id=c.id) recipients,(SELECT count(*) FROM legui_private.mail_outbox WHERE campaign_id=c.id AND delivered_at IS NOT NULL) delivered FROM legui_private.newsletter_campaigns c WHERE brand_id=? ORDER BY created_at DESC LIMIT 100)v`, configured, scope.BrandID))
		}
		if !validID(id) {
			return Result{}, notFound()
		}
		data, e := campaignData(h.repo.DB.WithContext(r.Context()), scope.BrandID, id)
		if e != nil {
			return Result{}, e
		}
		var out map[string]any
		json.Unmarshal(data, &out)
		var c struct{ BodyCipher, Segment string }
		q := h.repo.DB.WithContext(r.Context()).Raw(`SELECT body_cipher,segment FROM legui_private.newsletter_campaigns WHERE id=? AND brand_id=?`, id, scope.BrandID).Scan(&c)
		if q.Error != nil {
			return Result{}, q.Error
		}
		if q.RowsAffected != 1 {
			return Result{}, notFound()
		}
		body, e := auth.OpenCampaign(id, c.BodyCipher)
		if e != nil {
			return Result{}, e
		}
		rows, e := audience(h.repo.DB.WithContext(r.Context()), scope.BrandID, c.Segment)
		if e != nil {
			return Result{}, e
		}
		out["body"] = body
		out["eligible_count"] = len(rows)
		out["mail_configured"] = configured
		encoded, _ := json.Marshal(out)
		return Result{Data: encoded}, nil
	}
	if id == "" {
		input, e := decode[struct {
			Subject string `json:"subject"`
			Body    string `json:"body"`
			Segment string `json:"segment"`
		}](r)
		if e != nil {
			return Result{}, e
		}
		input.Subject = strings.TrimSpace(input.Subject)
		input.Body = strings.TrimSpace(input.Body)
		if len([]rune(input.Subject)) < 3 || len([]rune(input.Subject)) > 120 || strings.ContainsAny(input.Subject, "\r\n") || len([]rune(input.Body)) < 3 || len([]rune(input.Body)) > 4000 || !isOneOf(input.Segment, "all", "frequent", "recover", "new") {
			return Result{}, badInput("Revisá el asunto, mensaje y audiencia")
		}
		return h.mutate(r, p, scope, "campaign.create", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
			id := uuid.NewString()
			cipher, e := auth.SealCampaign(id, input.Body)
			if e != nil {
				return nil, 0, e
			}
			if e = tx.Exec(`INSERT INTO legui_private.newsletter_campaigns(id,brand_id,subject,body_cipher,segment) VALUES(?,?,?,?,?)`, id, scope.BrandID, input.Subject, cipher, input.Segment).Error; e != nil {
				return nil, 0, e
			}
			data, e := campaignData(tx, scope.BrandID, id)
			return data, 201, e
		})
	}
	if !validID(id) {
		return Result{}, notFound()
	}
	if _, e = decode[struct{}](r); e != nil {
		return Result{}, e
	}
	sending := strings.HasSuffix(r.URL.Path, "/send")
	return h.mutate(r, p, scope, "campaign.action", map[string]any{"id": id, "send": sending}, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var c struct{ State, Segment string }
		q := tx.Raw(`SELECT state,segment FROM legui_private.newsletter_campaigns WHERE id=? AND brand_id=? FOR UPDATE`, id, scope.BrandID).Scan(&c)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, 0, notFound()
		}
		if sending {
			if !configured {
				return nil, 0, conflict("Configurá SMTP y PUBLIC_SITE_URL antes de enviar correos")
			}
			if c.State != "draft" {
				return nil, 0, conflict("Esta campaña ya fue enviada o cancelada")
			}
			rows, e := audience(tx, scope.BrandID, c.Segment)
			if e != nil {
				return nil, 0, e
			}
			if len(rows) == 0 || len(rows) > 5000 {
				return nil, 0, conflict("La audiencia debe tener entre 1 y 5000 suscriptores confirmados")
			}
			snapshot := make([]map[string]string, 0, len(rows))
			for _, row := range rows {
				snapshot = append(snapshot, map[string]string{"id": row.ID, "email": row.Email})
			}
			encoded, _ := json.Marshal(snapshot)
			if e = tx.Exec(`INSERT INTO legui_private.newsletter_recipients(campaign_id,shopper_id,recipient_email) SELECT ?::uuid,r.id::uuid,r.email FROM jsonb_to_recordset(?::jsonb) AS r(id text,email text)`, id, string(encoded)).Error; e != nil {
				return nil, 0, e
			}
			if e = tx.Exec(`UPDATE legui_private.newsletter_campaigns SET state='queued',sent_at=clock_timestamp() WHERE id=?`, id).Error; e != nil {
				return nil, 0, e
			}
		} else {
			if c.State == "completed" {
				return nil, 0, conflict("La campaña ya terminó; los correos enviados no pueden retirarse")
			}
			if e := tx.Exec(`UPDATE legui_private.newsletter_campaigns SET state='canceled' WHERE id=?`, id).Error; e != nil {
				return nil, 0, e
			}
		}
		data, e := campaignData(tx, scope.BrandID, id)
		return data, 200, e
	})
}
func (h *Handler) newsletterUnsubscribe(r *http.Request) (Result, error) {
	if e := buyeraccess.Rate(r.Context(), h.repo.DB, r, "unsubscribe", 120); e != nil {
		return Result{}, buyerProblem(e)
	}
	in, e := decode[struct {
		Token string `json:"token"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	if !buyeraccess.ValidToken(in.Token) {
		return Result{}, notFound()
	}
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var row struct{ BrandID, ShopperID string }
		q := tx.Raw(`SELECT brand_id::text,shopper_id::text FROM legui_private.newsletter_unsubscribes WHERE digest=? AND expires_at>clock_timestamp()`, buyeraccess.Digest(in.Token)).Scan(&row)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return notFound()
		}
		updated := tx.Exec(`UPDATE legui_private.shopper_preferences SET newsletter=false,updated_at=clock_timestamp() WHERE shopper_id=? AND brand_id=? AND newsletter`, row.ShopperID, row.BrandID)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return nil
		}
		return tx.Exec(`INSERT INTO legui_private.shopper_preference_events(shopper_id,brand_id,newsletter,personalization) SELECT shopper_id,brand_id,newsletter,personalization FROM legui_private.shopper_preferences WHERE shopper_id=? AND brand_id=?`, row.ShopperID, row.BrandID).Error
	})
	return resultJSON(json.RawMessage(`{"unsubscribed":true}`), e)
}
func unsubscribeToken() (string, error) {
	b := make([]byte, 32)
	_, e := rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b), e
}
