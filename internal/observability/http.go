package observability

import (
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Log route patterns only: no query, authorization, body, email, raw URL or IP.
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-ID", id)
		started := time.Now()
		out := &statusWriter{ResponseWriter: w}
		defer func() {
			if recover() != nil {
				if out.status == 0 {
					http.Error(out, "No se pudo completar la solicitud", 500)
				}
				slog.Error("http panic", "request_id", id)
			}
			code := out.status
			if code == 0 {
				code = 200
			}
			slog.Info("http request", "request_id", id, "method", r.Method, "route", r.Pattern, "status", code, "duration_ms", time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(out, r)
	})
}
