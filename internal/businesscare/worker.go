package businesscare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

func StartWorker(ctx context.Context, db *gorm.DB) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				work, cancel := context.WithTimeout(ctx, 50*time.Second)
				if e := uploadOne(work, db); e != nil {
					_ = Record(work, db, "worker", "backup", "")
				}
				if e := cleanImage(work, db); e != nil {
					_ = Record(work, db, "worker", "image_cleanup", "")
				}
				_ = relayOne(work, db)
				cancel()
			}
		}
	}()
}
func cleanImage(ctx context.Context, db *gorm.DB) error {
	var item struct{ ObjectPath, Claim string }
	q := db.WithContext(ctx).Raw(`WITH picked AS(SELECT i.object_path FROM legui_private.staged_product_images i WHERE i.created_at<now()-interval '1 day' AND (i.lease_until IS NULL OR i.lease_until<now()) AND NOT EXISTS(SELECT 1 FROM public.products p WHERE p.image_url=i.public_url) ORDER BY i.created_at LIMIT 1 FOR UPDATE SKIP LOCKED) UPDATE legui_private.staged_product_images i SET claim=gen_random_uuid(),lease_until=now()+interval '10 minutes' FROM picked WHERE i.object_path=picked.object_path RETURNING i.object_path,i.claim::text`).Scan(&item)
	if q.Error != nil || q.RowsAffected == 0 {
		return q.Error
	}
	if !regexp.MustCompile(`^products/[0-9a-f-]{36}/[0-9a-f-]{36}\.(png|jpg)$`).MatchString(item.ObjectPath) {
		return errors.New("invalid cleanup path")
	}
	bucket := os.Getenv("SUPABASE_STORAGE_BUCKET")
	if bucket == "" {
		bucket = "catalog_images"
	}
	_, status, e := storage(ctx, "DELETE", bucket, item.ObjectPath, nil)
	if e != nil && status != 404 {
		return e
	}
	return db.WithContext(ctx).Exec(`DELETE FROM legui_private.staged_product_images WHERE object_path=? AND claim=? AND NOT EXISTS(SELECT 1 FROM public.products WHERE image_url=staged_product_images.public_url)`, item.ObjectPath, item.Claim).Error
}

// Only fixed, allowlisted categories leave the process. There is deliberately
// no HTTP body/header/query, user, email, URL, panic value or arbitrary stack.
func relayOne(ctx context.Context, db *gorm.DB) error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return nil
	}
	u, e := url.Parse(dsn)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User == nil {
		return errors.New("invalid monitoring configuration")
	}
	project := path.Base(u.Path)
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(project) {
		return errors.New("invalid monitoring project")
	}
	var row struct {
		Component, Code string
		Occurrences     int64
	}
	q := db.WithContext(ctx).Raw(`WITH picked AS(SELECT component,code FROM legui_private.runtime_errors WHERE occurrences>relayed_count AND (relay_lease_until IS NULL OR relay_lease_until<now()) ORDER BY last_seen_at LIMIT 1 FOR UPDATE SKIP LOCKED) UPDATE legui_private.runtime_errors e SET relay_lease_until=now()+interval '5 minutes' FROM picked p WHERE e.component=p.component AND e.code=p.code RETURNING e.component,e.code,e.occurrences`).Scan(&row)
	if q.Error != nil || q.RowsAffected == 0 {
		return q.Error
	}
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	now := time.Now().UTC().Format(time.RFC3339)
	event, _ := json.Marshal(map[string]any{"event_id": id, "timestamp": now, "platform": "other", "level": "error", "message": "LeguiBurger " + row.Component + " / " + row.Code, "fingerprint": []string{row.Component, row.Code}, "tags": map[string]string{"component": row.Component}, "extra": map[string]any{"occurrences": row.Occurrences}})
	header, _ := json.Marshal(map[string]any{"event_id": id, "dsn": dsn, "sent_at": now})
	item, _ := json.Marshal(map[string]any{"type": "event", "length": len(event)})
	payload := append(append(append(append(header, '\n'), item...), '\n'), event...)
	payload = append(payload, '\n')
	endpoint := *u
	endpoint.User = nil
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	endpoint.Path = path.Dir(u.Path) + "/api/" + project + "/envelope/"
	if path.Dir(u.Path) == "/" {
		endpoint.Path = "/api/" + project + "/envelope/"
	}
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewReader(payload))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/x-sentry-envelope")
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := client.Do(req)
	if e != nil {
		return errors.New("monitoring relay pending")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("monitoring relay pending")
	}
	return db.WithContext(ctx).Exec(`UPDATE legui_private.runtime_errors SET relayed_count=greatest(relayed_count,?) WHERE component=? AND code=?`, row.Occurrences, row.Component, row.Code).Error
}
