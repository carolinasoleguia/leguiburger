package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"leguiburger/internal/auth"
)

func JSONRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	t.Helper()

	var buf *bytes.Buffer
	switch v := body.(type) {
	case nil:
		buf = nil
	case []byte:
		buf = bytes.NewBuffer(v)
	case string:
		buf = bytes.NewBufferString(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("error al marshalar body JSON: %v", err)
		}
		buf = bytes.NewBuffer(b)
	}

	req := httptest.NewRequest(method, url, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func WithClaims(t *testing.T, req *http.Request, claims *auth.Claims) *http.Request {
	t.Helper()
	if claims == nil {
		return req
	}
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsKey, claims))
}

func DecodeJSONBody(t *testing.T, rr *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(out); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}
}

func AssertStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Fatalf("se esperaba status %d, se obtuvo %d", expected, rr.Code)
	}
}
