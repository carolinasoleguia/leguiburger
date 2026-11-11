package businesscare

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxBackup = 20 << 20

func backupCipher() (cipher.AEAD, error) {
	key := os.Getenv("BACKUP_ENCRYPTION_KEY")
	if key == "" {
		key = os.Getenv("AUTH_ENCRYPTION_KEY")
	}
	if key == "" {
		key = os.Getenv("JWT_SECRET")
	}
	if len(key) < 32 {
		return nil, errors.New("backup encryption requires a stable secret of at least 32 bytes")
	}
	sum := sha256.Sum256([]byte("leguiburger/private-backups/v1/" + key))
	block, e := aes.NewCipher(sum[:])
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(block)
}
func seal(body []byte, aad string) ([]byte, error) {
	a, e := backupCipher()
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, a.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return nil, e
	}
	return a.Seal(nonce, nonce, body, []byte(aad)), nil
}
func open(body []byte, aad string) ([]byte, error) {
	a, e := backupCipher()
	if e != nil {
		return nil, e
	}
	if len(body) < a.NonceSize() {
		return nil, errors.New("invalid backup")
	}
	return a.Open(nil, body[:a.NonceSize()], body[a.NonceSize():], []byte(aad))
}

// Storage credentials never follow redirects and provider response bodies never
// reach logs or browser errors. Object names come only from server-owned rows.
func storage(ctx context.Context, method, bucket, object string, body []byte) ([]byte, int, error) {
	base := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	key := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if base == "" || key == "" {
		return nil, 0, errors.New("private storage not configured")
	}
	target, e := url.JoinPath(base, "storage/v1/object", bucket, object)
	if e != nil {
		return nil, 0, e
	}
	mimeType := "application/octet-stream"
	if method == "DELETE" {
		target, e = url.JoinPath(base, "storage/v1/object", bucket)
		if e != nil {
			return nil, 0, e
		}
		body, _ = json.Marshal(map[string]any{"prefixes": []string{object}})
		mimeType = "application/json"
	}
	req, e := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if e != nil {
		return nil, 0, e
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("apikey", key)
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("x-upsert", "false")
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := client.Do(req)
	if e != nil {
		return nil, 0, e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Older Storage versions return HTTP 400 with a typed 404 in JSON.
		// Never classify invalid credentials or a network failure as absence.
		if resp.StatusCode == 400 {
			var failure struct {
				Code       string `json:"code"`
				StatusCode string `json:"statusCode"`
				Message    string `json:"message"`
			}
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			if json.Unmarshal(raw, &failure) == nil && (failure.Code == "NoSuchKey" || failure.StatusCode == "404" && failure.Message == "Object not found") {
				return nil, 404, errors.New("object not found")
			}
		}
		return nil, resp.StatusCode, errors.New("private storage unavailable")
	}
	if method != "GET" {
		return nil, resp.StatusCode, nil
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, maxBackup+4097))
	if len(data) > maxBackup+4096 {
		return nil, resp.StatusCode, errors.New("backup exceeds limit")
	}
	return data, resp.StatusCode, e
}
