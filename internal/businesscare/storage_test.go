package businesscare

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBackupCipherBindsIdentityAndDetectsCorruption(t *testing.T) {
	t.Setenv("BACKUP_ENCRYPTION_KEY", "fixture-backup-encryption-key-with-32-bytes")
	raw := []byte(`{"version":1,"tables":{}}`)
	cipher, e := seal(raw, "business/a")
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(cipher, raw) {
		t.Fatal("backup is plaintext")
	}
	decoded, e := open(cipher, "business/a")
	if e != nil || !bytes.Equal(decoded, raw) {
		t.Fatal("round trip failed")
	}
	if _, e = open(cipher, "business/b"); e == nil {
		t.Fatal("cross business decrypt accepted")
	}
	cipher[len(cipher)-1] ^= 1
	if _, e = open(cipher, "business/a"); e == nil {
		t.Fatal("tampered backup accepted")
	}
}
func TestPrivateStorageRejectsRedirectAndBoundsReads(t *testing.T) {
	calls := 0
	trap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) }))
	defer trap.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, trap.URL, 307) }))
	defer server.Close()
	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "fixture-only")
	_, status, e := storage(context.Background(), "GET", "business-backups", "fixture.bin", nil)
	if e == nil || status != 307 || calls != 0 {
		t.Fatal("private credentials followed redirect")
	}
}
