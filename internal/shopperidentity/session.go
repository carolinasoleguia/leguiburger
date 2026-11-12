// Package shopperidentity is consumer identity, separate from staff roles/JWT.
package shopperidentity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrSession = errors.New("shopper session unavailable")

const Cookie = "legui_shopper"

type Profile struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func Digest(raw string) string { s := sha256.Sum256([]byte(raw)); return hex.EncodeToString(s[:]) }
func Random() string {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic("entropy unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func Current(ctx context.Context, db *gorm.DB, r *http.Request) (Profile, error) {
	return current(ctx, db, r, false)
}
func Lock(ctx context.Context, db *gorm.DB, r *http.Request) (Profile, error) {
	return current(ctx, db, r, true)
}
func current(ctx context.Context, db *gorm.DB, r *http.Request, lock bool) (Profile, error) {
	var p Profile
	if !Origin(r) {
		return p, ErrSession
	}
	c, e := r.Cookie(Cookie)
	if e != nil || len(c.Value) != 43 {
		return p, ErrSession
	}
	sql := `SELECT p.id::text,p.name,p.email,p.email_verified FROM legui_private.shopper_sessions s JOIN legui_private.shoppers p ON p.id=s.shopper_id AND p.active WHERE s.digest=? AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp()`
	if lock {
		sql += ` FOR SHARE OF s,p`
	}
	q := db.WithContext(ctx).Raw(sql, Digest(c.Value)).Scan(&p)
	if q.Error != nil {
		return p, q.Error
	}
	if q.RowsAffected != 1 {
		return p, ErrSession
	}
	return p, nil
}
func SetCookie(w http.ResponseWriter, raw string) {
	max := 7 * 86400
	if raw == "" {
		max = -1
	}
	http.SetCookie(w, &http.Cookie{Name: Cookie, Value: raw, Path: "/api", HttpOnly: true, Secure: os.Getenv("APP_ENV") == "production" || strings.HasPrefix(os.Getenv("PUBLIC_SITE_URL"), "https://"), SameSite: http.SameSiteLaxMode, MaxAge: max, Expires: time.Now().Add(time.Duration(max) * time.Second)})
}
func Origin(r *http.Request) bool {
	site := r.Header.Get("Sec-Fetch-Site")
	if site == "same-site" || site == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if site := os.Getenv("PUBLIC_SITE_URL"); site != "" {
		return origin == strings.TrimRight(site, "/")
	}
	u, e := url.Parse(origin)
	return e == nil && os.Getenv("APP_ENV") != "production" && u.Scheme == "http" && (u.Host == r.Host || u.Host == "127.0.0.1:5173" || u.Host == "localhost:5173")
}
