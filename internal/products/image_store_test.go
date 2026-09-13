package products

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseProductImageStore_SaveProductImage(t *testing.T) {
	var receivedPath string
	var receivedAuth string
	var receivedAPIKey string
	var receivedUpsert string
	var receivedContentType string
	var receivedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")
		receivedAPIKey = r.Header.Get("apikey")
		receivedUpsert = r.Header.Get("x-upsert")
		receivedContentType = r.Header.Get("Content-Type")

		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		receivedBody = buf.String()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileHeader := multipartFileHeader(t, "burger.png", "fake-image-bytes")
	store := &SupabaseProductImageStore{
		BaseURL:    server.URL,
		APIKey:     "service-key",
		Bucket:     "product-images",
		HTTPClient: server.Client(),
	}

	imageURL, err := store.SaveProductImage(context.Background(), fileHeader)
	if err != nil {
		t.Fatalf("no se esperaba error al subir imagen: %v", err)
	}

	if !strings.HasPrefix(receivedPath, "/storage/v1/object/product-images/products/") || !strings.HasSuffix(receivedPath, ".png") {
		t.Fatalf("path de upload inesperado: %s", receivedPath)
	}
	if receivedAuth != "Bearer service-key" || receivedAPIKey != "service-key" {
		t.Fatalf("headers de autenticacion inesperados: auth=%q apikey=%q", receivedAuth, receivedAPIKey)
	}
	if receivedUpsert != "true" {
		t.Fatalf("se esperaba x-upsert=true, se obtuvo %q", receivedUpsert)
	}
	if receivedContentType != "image/png" {
		t.Fatalf("content-type inesperado: %s", receivedContentType)
	}
	if receivedBody != "fake-image-bytes" {
		t.Fatalf("body inesperado: %s", receivedBody)
	}
	if !strings.HasPrefix(imageURL, server.URL+"/storage/v1/object/public/product-images/products/") || !strings.HasSuffix(imageURL, ".png") {
		t.Fatalf("URL publica inesperada: %s", imageURL)
	}
}

func multipartFileHeader(t *testing.T, fileName, content string) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image_file", fileName)
	if err != nil {
		t.Fatalf("no se pudo crear el archivo multipart: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("no se pudo escribir el archivo multipart: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("no se pudo cerrar el multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(10 << 20); err != nil {
		t.Fatalf("no se pudo parsear multipart: %v", err)
	}

	_, fileHeader, err := req.FormFile("image_file")
	if err != nil {
		t.Fatalf("no se pudo obtener file header: %v", err)
	}

	return fileHeader
}
