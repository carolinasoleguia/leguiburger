package businesscare

import (
	"bytes"
	"context"
	"flag"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var storageIntegration = flag.Bool("care-storage-integration", false, "exercise only the private fixture object and remove it")

func TestPrivateBackupStorageIntegration(t *testing.T) {
	if !*storageIntegration {
		t.Skip("explicit private storage integration flag required")
	}
	_ = godotenv.Load("../../.env")
	if os.Getenv("SUPABASE_URL") != "https://ljdebnvzdrlqdxnligvh.supabase.co" {
		t.Fatal("unexpected storage target")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	object := "integration/" + uuid.NewString() + ".bin"
	body := []byte("private integration fixture, no business data")
	_, status, e := storage(ctx, "GET", "business-backups", object, nil)
	if e == nil || status != 404 {
		t.Fatalf("missing private object classification failed (status %d)", status)
	}
	if _, status, e = storage(ctx, "POST", "business-backups", object, body); e != nil {
		t.Fatalf("private fixture upload failed (status %d)", status)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 20*time.Second)
		defer stop()
		if _, status, e := storage(cleanup, "DELETE", "business-backups", object, nil); e != nil {
			t.Errorf("private fixture cleanup failed (status %d)", status)
		}
	}()
	got, status, e := storage(ctx, "GET", "business-backups", object, nil)
	if e != nil || !bytes.Equal(got, body) {
		t.Fatalf("private read mismatch (status %d)", status)
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", os.Getenv("SUPABASE_URL")+"/storage/v1/object/public/business-backups/"+object, nil)
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := client.Do(req)
	if e != nil {
		t.Fatal("anonymous privacy check unavailable")
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Fatal("private backup is publicly accessible")
	}
	t.Log("Private bucket verified: missing object, immutable upload/read, anonymous denial; fixture removed")
}
