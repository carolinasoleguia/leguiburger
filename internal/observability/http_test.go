package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIdentityAndPanicIsolation(t *testing.T) {
	handler := HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("sensitive panic text") }))
	first, second := httptest.NewRecorder(), httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest("GET", "/test?token=private", nil))
	handler.ServeHTTP(second, httptest.NewRequest("GET", "/test", nil))
	if first.Code != 500 || first.Header().Get("X-Request-ID") == "" || first.Header().Get("X-Request-ID") == second.Header().Get("X-Request-ID") {
		t.Fatal("panic or request identity contract failed")
	}
	if first.Body.String() == "sensitive panic text" {
		t.Fatal("panic disclosed")
	}
}
