package operations

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/auth"
	"leguiburger/internal/brands"
	"leguiburger/internal/db"
	"leguiburger/internal/employees"
	"leguiburger/internal/products"
	"leguiburger/internal/tenants"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

var securityIntegration = flag.Bool("security-integration", false, "Run rollback-only identity/permissions HTTP integration against the configured database")

func TestSecurityIntegration(t *testing.T) {
	if !*securityIntegration {
		t.Skip("opt-in security integration")
	}
	_ = godotenv.Load("../../.env")
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: os.Getenv("DATABASE_URL"), PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("database connection unavailable")
	}
	pool, _ := database.DB()
	defer pool.Close()
	if err = VerifyDatabase(database, os.Getenv("DATABASE_URL")); err != nil {
		t.Fatal(err)
	}
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	previous := db.DB
	db.DB = tx
	defer func() { db.DB = previous }()
	t.Setenv("JWT_SECRET", "security-integration-fixture-not-used-by-running-app")
	t.Setenv("AUTH_ENCRYPTION_KEY", "security-integration-encryption-fixture")
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_REQUIRE_PLATFORM_MFA", "true")
	t.Setenv("PUBLIC_SITE_URL", "http://127.0.0.1:5173")
	must := func(q string, args ...any) {
		t.Helper()
		if e := tx.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	b1, b2, t1, t2, owner, manager, employee := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	password := "fixture-passphrase-123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	email := func(id string) string { return id + "@example.invalid" }
	must(`INSERT INTO public.brands(id,name) VALUES(?,'__security_http_a'),(?,'__security_http_b')`, b1, b2)
	must(`INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(?,?,'__security_http_a'),(?,?,'__security_http_b')`, t1, b1, t2, b2)
	must(`INSERT INTO public.users(id,first_name,last_name,email,password_hash,role,brand_id,is_active) VALUES(?,'Owner','Fixture',?,?,'owner',NULL,true),(?,'Manager','Fixture',?,?,'admin',?,true)`, owner, email(owner), string(hashed), manager, email(manager), string(hashed), b1)
	must(`INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active) VALUES(?,?,'Employee','Fixture',?,?,'employee',true)`, employee, t1, email(employee), string(hashed))
	credentialService, e := auth.NewService(auth.NewRepository(), tenants.NewRepository())
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	auth.RegisterSecurityRoutes(mux, tx, credentialService)
	RegisterRoutes(mux, tx)
	tenantHandler := tenants.NewHandler(tenants.NewService(tenants.NewRepository(), brands.NewRepository()))
	employeeHandler := employees.NewHandler(employees.NewService(employees.NewRepository(), tenants.NewRepository()))
	productHandler := products.NewHandler(products.NewService(products.NewRepository(), brands.NewRepository()))
	for prefix, handler := range map[string]http.HandlerFunc{"/api/tenants": tenantHandler.HandleTenantRoutes, "/api/employees": employeeHandler.HandleEmployeeRoutes, "/api/products": productHandler.HandleProductRoutes} {
		wrapped := auth.AuthMiddlewareWithDB(tx, auth.AuthorizeLegacy(handler))
		mux.HandleFunc(prefix, wrapped)
		mux.HandleFunc(prefix+"/", wrapped)
	}
	type reply struct {
		body   map[string]any
		cookie *http.Cookie
		status int
	}
	request := func(method, path, token string, cookie *http.Cookie, headers map[string]string, body any, expected int) reply {
		t.Helper()
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		r.RemoteAddr = "192.0.2.10:1234"
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		for key, value := range headers {
			r.Header.Set(key, value)
		}
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, r)
		var payload map[string]any
		_ = json.Unmarshal(recorder.Body.Bytes(), &payload)
		if recorder.Code != expected {
			t.Fatalf("%s %s expected %d got %d code=%v", method, path, expected, recorder.Code, payload["code"])
		}
		var updated *http.Cookie
		for _, item := range recorder.Result().Cookies() {
			if item.Name == "legui_refresh" {
				updated = item
			}
		}
		return reply{body: payload, cookie: updated, status: recorder.Code}
	}
	login := func(id string) reply {
		t.Helper()
		return request("POST", "/api/auth/login", "", nil, nil, map[string]any{"email": email(id), "password": password}, 200)
	}
	tokenOf := func(result reply) string {
		t.Helper()
		value, ok := result.body["token"].(string)
		if !ok || value == "" {
			t.Fatal("missing access token")
		}
		return value
	}
	for _, path := range []string{"/api/tenants", "/api/employees", "/api/products", "/api/orders", "/api/audit", "/api/team/invitations", "/api/auth/me", "/api/auth/sessions"} {
		request("GET", path, "", nil, nil, nil, 401)
	}
	staff := login(employee)
	staffToken := tokenOf(staff)
	claims, e := auth.ValidateToken(staffToken)
	if e != nil || claims.SessionID == "" || claims.Purpose != "access" || time.Until(claims.ExpiresAt.Time) > 16*time.Minute {
		t.Fatal("invalid session/access token contract")
	}
	if staff.cookie == nil || !staff.cookie.HttpOnly || staff.cookie.Path != "/api/auth" || staff.cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("invalid refresh cookie contract")
	}
	request("PUT", "/api/employees/"+employee, staffToken, nil, nil, map[string]any{"role": "owner"}, 403)
	request("PUT", "/api/tenants/"+t2, staffToken, nil, nil, map[string]any{"active": false}, 403)
	request("DELETE", "/api/tenants/"+t2, staffToken, nil, nil, nil, 403)
	request("GET", "/api/employees", staffToken, nil, map[string]string{"X-Tenant-ID": t2}, nil, 403)
	request("GET", "/api/products", staffToken, nil, map[string]string{"X-Brand-ID": b2}, nil, 403)
	request("POST", "/api/products", staffToken, nil, nil, map[string]any{}, 403)
	request("GET", "/api/products", staffToken, nil, nil, nil, 200)
	request("GET", "/api/audit", staffToken, nil, nil, nil, 403)
	request("GET", "/api/analytics/platform", staffToken, nil, nil, nil, 403)
	managerLogin := login(manager)
	managerToken := tokenOf(managerLogin)
	request("GET", "/api/employees", managerToken, nil, map[string]string{"X-Tenant-ID": t2}, nil, 403)
	request("PUT", "/api/employees/"+employee, managerToken, nil, map[string]string{"X-Tenant-ID": t1}, map[string]any{"role": "owner"}, 400)
	request("POST", "/api/tenants", managerToken, nil, nil, map[string]any{}, 403)
	request("GET", "/api/analytics/platform", managerToken, nil, nil, nil, 403)
	request("GET", "/api/audit", managerToken, nil, nil, nil, 200)
	request("POST", "/api/auth/refresh", "", staff.cookie, map[string]string{"Origin": "https://evil.example"}, map[string]any{}, 403)
	request("POST", "/api/auth/refresh", "", staff.cookie, map[string]string{"X-Legui-Account": owner}, map[string]any{}, 409)
	refreshed := request("POST", "/api/auth/refresh", "", staff.cookie, nil, map[string]any{}, 200)
	if refreshed.cookie == nil || refreshed.cookie.Value == staff.cookie.Value {
		t.Fatal("refresh credential did not rotate")
	}
	request("POST", "/api/auth/refresh", "", staff.cookie, nil, map[string]any{}, 401)
	request("GET", "/api/auth/me", tokenOf(refreshed), nil, nil, nil, 401)
	// Password changes revoke previously issued access and refresh tokens.
	staff = login(employee)
	staffToken = tokenOf(staff)
	must(`UPDATE public.employees SET password_hash=? WHERE id=?`, string(hashed)+"changed", employee)
	request("GET", "/api/auth/me", staffToken, nil, nil, nil, 401)
	request("POST", "/api/auth/refresh", "", staff.cookie, nil, map[string]any{}, 401)
	must(`UPDATE public.employees SET password_hash=? WHERE id=?`, string(hashed), employee)
	// Real password recovery: generic response, encrypted delivery, single-use token.
	forgot := request("POST", "/api/auth/password/forgot", "", nil, nil, map[string]any{"email": email(employee)}, 200)
	unknown := request("POST", "/api/auth/password/forgot", "", nil, nil, map[string]any{"email": "unknown-security-fixture@example.invalid"}, 200)
	if forgot.body["message"] != unknown.body["message"] || forgot.body["token"] != nil {
		t.Fatal("recovery leaks account/token information")
	}
	var encrypted string
	if e = tx.Raw(`SELECT body_cipher FROM legui_private.mail_outbox WHERE recipient=? ORDER BY created_at DESC LIMIT 1`, email(employee)).Scan(&encrypted).Error; e != nil {
		t.Fatal(e)
	}
	recoveryBody := decryptFixtureMail(t, encrypted, email(employee))
	lines := strings.Split(recoveryBody, "\n")
	var resetToken string
	for _, line := range lines {
		if strings.HasPrefix(line, "http://") {
			u, _ := url.Parse(line)
			values, _ := url.ParseQuery(u.Fragment)
			resetToken = values.Get("token")
		}
	}
	if resetToken == "" || strings.Contains(encrypted, resetToken) {
		t.Fatal("recovery token delivery/encryption invalid")
	}
	newPassword := "new-security-passphrase-123!"
	request("POST", "/api/auth/password/reset", "", nil, nil, map[string]any{"token": resetToken, "password": newPassword}, 200)
	request("POST", "/api/auth/password/reset", "", nil, nil, map[string]any{"token": resetToken, "password": newPassword}, 400)
	request("POST", "/api/auth/login", "", nil, nil, map[string]any{"email": email(employee), "password": newPassword}, 200)
	// Invitations stay scoped to the inviter and cannot assign elevated roles.
	invitedEmail := email(uuid.NewString())
	request("POST", "/api/team/invitations", managerToken, nil, nil, map[string]any{"email": invitedEmail, "tenant_id": t2, "role": "cashier"}, 403)
	request("POST", "/api/team/invitations", managerToken, nil, nil, map[string]any{"email": invitedEmail, "tenant_id": t1, "role": "owner"}, 400)
	invitation := request("POST", "/api/team/invitations", managerToken, nil, nil, map[string]any{"email": invitedEmail, "tenant_id": t1, "role": "cashier"}, 201)
	invitationURL, _ := url.Parse(invitation.body["invitation_url"].(string))
	values, _ := url.ParseQuery(invitationURL.Fragment)
	invitationToken := values.Get("token")
	acceptance := map[string]any{"token": invitationToken, "password": newPassword, "first_name": "Invited", "last_name": "Fixture"}
	request("POST", "/api/auth/invitations/accept", "", nil, nil, acceptance, 201)
	request("POST", "/api/auth/invitations/accept", "", nil, nil, acceptance, 400)
	var invitationCount int64
	tx.Raw(`SELECT count(*) FROM public.employees WHERE email=? AND role='cashier' AND tenant_id=?`, invitedEmail, t1).Scan(&invitationCount)
	if invitationCount != 1 {
		t.Fatal("invitation did not create scoped collaborator")
	}
	// MFA enrollment is required before a platform account can access any API.
	challenge := login(owner)
	challengeToken := challenge.body["challenge"].(string)
	if challenge.body["mfa_required"] != true || challenge.body["token"] != nil {
		t.Fatal("platform password bypassed MFA")
	}
	request("GET", "/api/analytics/platform", challengeToken, nil, nil, nil, 401)
	enrollment := request("POST", "/api/auth/mfa/enroll", challengeToken, nil, nil, map[string]any{}, 200)
	secret := enrollment.body["secret"].(string)
	initialCode := fixtureTOTP(secret)
	verified := request("POST", "/api/auth/mfa/verify", challengeToken, nil, nil, map[string]any{"code": initialCode}, 200)
	ownerToken := tokenOf(verified)
	codes := verified.body["recovery_codes"].([]any)
	if len(codes) != 10 {
		t.Fatal("missing one-use recovery codes")
	}
	request("GET", "/api/analytics/platform", ownerToken, nil, nil, nil, 200)
	request("POST", "/api/auth/mfa/verify", challengeToken, nil, nil, map[string]any{"code": initialCode}, 401)
	secondChallenge := login(owner).body["challenge"].(string)
	request("POST", "/api/auth/mfa/verify", secondChallenge, nil, nil, map[string]any{"code": initialCode}, 401)
	request("POST", "/api/auth/mfa/verify", secondChallenge, nil, nil, map[string]any{"recovery_code": codes[0]}, 200)
	thirdChallenge := login(owner).body["challenge"].(string)
	request("POST", "/api/auth/mfa/verify", thirdChallenge, nil, nil, map[string]any{"recovery_code": codes[0]}, 401)
	request("POST", "/api/auth/logout", "", verified.cookie, nil, map[string]any{}, 200)
	request("GET", "/api/analytics/platform", ownerToken, nil, nil, nil, 401)
	t.Setenv("AUTH_REQUIRE_PLATFORM_MFA", "false")
	passwordOnly := login(owner)
	passwordOnlyToken := tokenOf(passwordOnly)
	passwordClaims, e := auth.ValidateToken(passwordOnlyToken)
	if e != nil || passwordClaims.MFA || passwordOnly.body["mfa_required"] == true || passwordOnly.cookie == nil {
		t.Fatal("password-only login unexpectedly requires MFA")
	}
	request("GET", "/api/analytics/platform", passwordOnlyToken, nil, nil, nil, 200)
	passwordRefreshed := request("POST", "/api/auth/refresh", "", passwordOnly.cookie, nil, map[string]any{}, 200)
	request("GET", "/api/analytics/platform", tokenOf(passwordRefreshed), nil, nil, nil, 200)
	t.Setenv("AUTH_REQUIRE_PLATFORM_MFA", "true")
	request("GET", "/api/analytics/platform", tokenOf(passwordRefreshed), nil, nil, nil, 401)
	request("POST", "/api/auth/refresh", "", passwordRefreshed.cookie, nil, map[string]any{}, 401)
	request("GET", "/api/auth/sessions", managerToken, nil, nil, nil, 200)
	// Email changes require password reauthentication and both mailbox capabilities.
	t.Setenv("SMTP_HOST", "smtp.example.invalid")
	t.Setenv("SMTP_FROM", "noreply@example.invalid")
	emailActor := uuid.NewString()
	must(`INSERT INTO public.users(id,first_name,last_name,email,password_hash,role,brand_id,is_active) VALUES(?,'Email','Fixture',?,?,'admin',?,true)`, emailActor, email(emailActor), string(hashed), b1)
	emailLogin := login(emailActor)
	emailToken := tokenOf(emailLogin)
	newEmail := emailActor + "-new@example.invalid"
	request("POST", "/api/auth/email/change", emailToken, nil, nil, map[string]any{"email": newEmail, "current_password": "incorrect"}, 401)
	request("POST", "/api/auth/email/change", emailToken, nil, nil, map[string]any{"email": newEmail, "current_password": password}, 202)
	mailToken := func(recipient string) string {
		t.Helper()
		var cipherText string
		if e := tx.Raw(`SELECT body_cipher FROM legui_private.mail_outbox WHERE recipient=? ORDER BY created_at DESC LIMIT 1`, recipient).Scan(&cipherText).Error; e != nil {
			t.Fatal(e)
		}
		for _, line := range strings.Split(decryptFixtureMail(t, cipherText, recipient), "\n") {
			if strings.HasPrefix(line, "http://") {
				u, _ := url.Parse(line)
				values, _ := url.ParseQuery(u.Fragment)
				return values.Get("token")
			}
		}
		t.Fatal("email confirmation capability missing")
		return ""
	}
	oldApproval, newApproval := mailToken(email(emailActor)), mailToken(newEmail)
	if oldApproval == newApproval {
		t.Fatal("email confirmations must be independent")
	}
	firstApproval := request("POST", "/api/auth/email/confirm", "", nil, nil, map[string]any{"token": newApproval}, 200)
	if firstApproval.body["completed"] != false {
		t.Fatal("one mailbox changed the login email")
	}
	request("GET", "/api/auth/me", emailToken, nil, nil, nil, 200)
	completedEmail := request("POST", "/api/auth/email/confirm", "", nil, nil, map[string]any{"token": oldApproval}, 200)
	if completedEmail.body["completed"] != true {
		t.Fatal("both approvals did not complete email change")
	}
	request("GET", "/api/auth/me", emailToken, nil, nil, nil, 401)
	request("POST", "/api/auth/email/confirm", "", nil, nil, map[string]any{"token": oldApproval}, 400)
	request("POST", "/api/auth/login", "", nil, nil, map[string]any{"email": newEmail, "password": password}, 200)
	request("POST", "/api/auth/logout-all", managerToken, managerLogin.cookie, nil, map[string]any{}, 200)
	request("GET", "/api/auth/me", managerToken, nil, nil, nil, 401)
	// Limits apply to failed requests and survive handler instances.
	for index := 0; index < 10; index++ {
		request("POST", "/api/auth/login", "", nil, nil, map[string]any{"email": "rate-limit-fixture@example.invalid", "password": password}, 401)
	}
	request("POST", "/api/auth/login", "", nil, nil, map[string]any{"email": "rate-limit-fixture@example.invalid", "password": password}, 429)
	// No fixture, token, factor, invitation or outbox message survives this test.
	if e = tx.WithContext(context.Background()).Rollback().Error; e != nil {
		t.Fatal(e)
	}
	t.Log("Security HTTP integration passed: tenant/role boundaries, MFA/replay, refresh rotation/reuse, logout, password reset, invitations, audit, cookies and rate limiting; all fixtures rolled back")
}
func fixtureTOTP(secret string) string {
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(time.Now().Unix()/30))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[offset:offset+4])&0x7fffffff)%1000000)
}
func decryptFixtureMail(t *testing.T, encrypted, email string) string {
	t.Helper()
	key := sha256.Sum256([]byte("leguiburger/auth-secrets/v1/security-integration-encryption-fixture"))
	block, _ := aes.NewCipher(key[:])
	aead, _ := cipher.NewGCM(block)
	raw, e := base64.RawURLEncoding.DecodeString(encrypted)
	if e != nil || len(raw) < aead.NonceSize() {
		t.Fatal("invalid encrypted mail")
	}
	body, e := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("mail/"+email))
	if e != nil {
		t.Fatal("cannot decrypt fixture mail")
	}
	return string(body)
}
