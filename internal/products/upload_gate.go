package products

import (
	"leguiburger/internal/auth"
	"net/http"
	"strings"
)

var imageUploadSlots = make(chan struct{}, 2)

// Acquire before opening the legacy mutation transaction. This bounds decoder
// memory and leaves pool connections for the independent upload journal.
func LimitImageUploads(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
			select {
			case imageUploadSlots <- struct{}{}:
				defer func() { <-imageUploadSlots }()
			default:
				auth.RespondWithError(w, 429, "IMAGE_UPLOAD_BUSY", "Hay otras imágenes subiendo. Reintentá en unos segundos.")
				return
			}
		}
		next(w, r)
	}
}
