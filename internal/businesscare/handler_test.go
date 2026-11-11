package businesscare

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEveryCareEndpointRejectsAnonymousRequests(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, nil)
	for _, route := range []string{"POST /api/care/account-close", "GET /api/care/support", "POST /api/care/support", "GET /api/care/support/a", "POST /api/care/support/a/messages", "POST /api/care/support/a/close", "GET /api/care/backups", "POST /api/care/backups", "GET /api/care/backups/a/download", "GET /api/care/closure", "POST /api/care/closure", "GET /api/care/errors", "POST /api/care/errors"} {
		parts := strings.SplitN(route, " ", 2)
		r := httptest.NewRequest(parts[0], parts[1], strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Errorf("%s accepted anonymous request: %d", route, w.Code)
		}
	}
}
