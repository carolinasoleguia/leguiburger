package buyeraccess

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type BrowserSession struct{ Secret, Key string }

func CookieName(tenant string) string {
	v, e := uuid.Parse(tenant)
	if e != nil || v == uuid.Nil {
		return ""
	}
	return "legui_buyer_" + strings.ReplaceAll(v.String(), "-", "")
}
func SameOrigin(r *http.Request) bool {
	site := r.Header.Get("Sec-Fetch-Site")
	if site == "cross-site" || site == "same-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if configured := os.Getenv("PUBLIC_SITE_URL"); configured != "" {
		return origin == strings.TrimRight(configured, "/")
	}
	u, e := url.Parse(origin)
	if e != nil {
		return false
	}
	if os.Getenv("APP_ENV") == "production" {
		return false
	}
	return u.Scheme == "http" && (u.Host == r.Host || u.Host == "127.0.0.1:5173" || u.Host == "localhost:5173")
}
func CookieSession(r *http.Request, tenant string) (BrowserSession, bool) {
	var s BrowserSession
	if !SameOrigin(r) {
		return s, false
	}
	name := CookieName(tenant)
	if name == "" {
		return s, false
	}
	c, e := r.Cookie(name)
	if e != nil {
		return s, false
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 2 || !ValidToken(parts[0]) {
		return s, false
	}
	key, e := uuid.Parse(parts[1])
	if e != nil || key == uuid.Nil {
		return s, false
	}
	return BrowserSession{Secret: parts[0], Key: key.String()}, true
}
func NewBrowserSession() BrowserSession {
	var raw [32]byte
	if _, e := rand.Read(raw[:]); e != nil {
		panic("buyer entropy unavailable")
	}
	return BrowserSession{Secret: base64.RawURLEncoding.EncodeToString(raw[:]), Key: uuid.NewString()}
}
func SetCookie(w http.ResponseWriter, tenant string, s BrowserSession) {
	http.SetCookie(w, &http.Cookie{Name: CookieName(tenant), Value: s.Secret + "." + s.Key, Path: "/api/public", HttpOnly: true, Secure: os.Getenv("APP_ENV") == "production" || strings.HasPrefix(os.Getenv("PUBLIC_SITE_URL"), "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 86400, Expires: time.Now().Add(24 * time.Hour)})
}
func PaymentKey(s BrowserSession) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("legui-payment:"+s.Secret)).String()
}
