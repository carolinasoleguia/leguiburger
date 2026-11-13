// Package buyeraccess validates a guest capability. It never creates an app session.
package buyeraccess

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/requestguard"
	"leguiburger/internal/shopperidentity"
	"net/http"
	"strings"
)

var ErrInvalid = errors.New("invalid buyer capability")
var ErrRateLimited = errors.New("buyer request rate exceeded")

type Grant struct{ OrderID, TenantID, BrandID string }

func Digest(raw string) string { sum := sha256.Sum256([]byte(raw)); return hex.EncodeToString(sum[:]) }
func Token(r *http.Request) string {
	if r.Header.Get("Authorization") != "" {
		return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	tenant := r.Header.Get("X-Buyer-Tenant")
	if tenant == "" {
		tenant = r.PathValue("id")
	}
	if session, ok := CookieSession(r, tenant); ok {
		return session.Secret
	}
	return ""
}
func ValidToken(raw string) bool {
	decoded, e := base64.RawURLEncoding.DecodeString(raw)
	return e == nil && len(decoded) == 32 && len(raw) == 43
}
func TrackingToken(raw string) string {
	sum := sha256.Sum256([]byte("leguiburger:buyer-tracking:" + raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
func Resolve(ctx context.Context, db *gorm.DB, raw string, lock bool) (Grant, error) {
	var grant Grant
	if !ValidToken(raw) {
		return grant, ErrInvalid
	}
	query := `SELECT a.order_id::text,a.tenant_id::text,t.brand_id::text FROM legui_private.buyer_orders a JOIN public.orders o ON o.id=a.order_id AND o.tenant_id=a.tenant_id AND o.payment_environment='production' JOIN public.tenants t ON t.id=a.tenant_id AND t.active JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL AND b.closed_at IS NULL WHERE a.access_digest=? AND a.expires_at>clock_timestamp()`
	if lock {
		query += ` FOR SHARE OF a,t,b`
	}
	q := db.WithContext(ctx).Raw(query, Digest(raw)).Scan(&grant)
	if q.Error != nil {
		return grant, q.Error
	}
	if q.RowsAffected != 1 {
		return grant, ErrInvalid
	}
	return grant, nil
}

// Distributed counters survive restarts and coordinate all API replicas.
func Rate(ctx context.Context, db *gorm.DB, r *http.Request, scope string, max int) error {
	return RateIdentity(ctx, db, scope, requestguard.ClientIP(r), max)
}
func RateIdentity(ctx context.Context, db *gorm.DB, scope, identity string, max int) error {
	var count int
	q := db.WithContext(ctx).Raw(`INSERT INTO legui_private.auth_rate_limits(bucket,attempts) VALUES(?,1) ON CONFLICT(bucket) DO UPDATE SET attempts=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN 1 ELSE auth_rate_limits.attempts+1 END,window_start=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN clock_timestamp() ELSE auth_rate_limits.window_start END RETURNING attempts`, "buyer:"+scope+":"+Digest(identity)).Scan(&count)
	if q.Error != nil {
		return q.Error
	}
	if count > max {
		return ErrRateLimited
	}
	return nil
}

// ResolveRequest accepts either a browser capability or an authenticated owner.
// An explicit account-order header never falls back to a guest cookie.
func ResolveRequest(ctx context.Context, db *gorm.DB, r *http.Request, lock bool) (Grant, error) {
	if id := r.Header.Get("X-Shopper-Order"); id != "" {
		if _, e := uuid.Parse(id); e != nil {
			return Grant{}, ErrInvalid
		}
		p, e := shopperidentity.Current(ctx, db, r)
		if lock {
			p, e = shopperidentity.Lock(ctx, db, r)
		}
		if e != nil {
			return Grant{}, ErrInvalid
		}
		var g Grant
		sql := `SELECT o.id::text order_id,o.tenant_id::text,t.brand_id::text FROM legui_private.shopper_orders a JOIN public.orders o ON o.id=a.order_id AND o.payment_environment='production' JOIN public.tenants t ON t.id=o.tenant_id AND t.active JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL AND b.closed_at IS NULL WHERE a.shopper_id=? AND o.id=?`
		if lock {
			sql += ` FOR SHARE OF a,t,b`
		}
		q := db.WithContext(ctx).Raw(sql, p.ID, id).Scan(&g)
		if q.Error != nil {
			return g, q.Error
		}
		if q.RowsAffected != 1 {
			return g, ErrInvalid
		}
		return g, nil
	}
	return Resolve(ctx, db, Token(r), lock)
}
