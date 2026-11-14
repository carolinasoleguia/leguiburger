package products

import (
	"bytes"
	"context"
	"image"
	"image/png"
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

	var pngBody bytes.Buffer
	if err := png.Encode(&pngBody, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	fileHeader := multipartFileHeader(t, "burger.svg", pngBody.String()+"<script>private metadata</script>")
	store := &SupabaseProductImageStore{
		BaseURL:    server.URL,
		APIKey:     "service-key",
		Bucket:     "product-images",
		HTTPClient: server.Client(),
	}

	imageURL, err := store.SaveProductImage(withImageBrand(context.Background(), "00000000-0000-4000-8000-000000000001"), fileHeader)
	if err != nil {
		t.Fatalf("no se esperaba error al subir imagen: %v", err)
	}

	if !strings.HasPrefix(receivedPath, "/storage/v1/object/product-images/products/") || !strings.HasSuffix(receivedPath, ".png") {
		t.Fatalf("path de upload inesperado: %s", receivedPath)
	}
	if receivedAuth != "Bearer service-key" || receivedAPIKey != "service-key" {
		t.Fatalf("headers de autenticacion inesperados: auth=%q apikey=%q", receivedAuth, receivedAPIKey)
	}
	if receivedUpsert != "false" {
		t.Fatalf("se esperaba x-upsert=false, se obtuvo %q", receivedUpsert)
	}
	if receivedContentType != "image/png" {
		t.Fatalf("content-type inesperado: %s", receivedContentType)
	}
	if strings.Contains(receivedBody, "script") {
		t.Fatal("untrusted trailing content was not stripped")
	}
	if _, err := png.Decode(strings.NewReader(receivedBody)); err != nil {
		t.Fatal("invalid rewritten image")
	}
	if !strings.HasPrefix(imageURL, server.URL+"/storage/v1/object/public/product-images/products/") || !strings.HasSuffix(imageURL, ".png") {
		t.Fatalf("URL publica inesperada: %s", imageURL)
	}
}

func TestImagesRejectInvalidContentAndRedirects(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "https://example.invalid/steal", 307)
	}))
	defer server.Close()
	store := &SupabaseProductImageStore{BaseURL: server.URL, APIKey: "private-fixture", Bucket: "images", HTTPClient: server.Client()}
	ctx := withImageBrand(context.Background(), "00000000-0000-4000-8000-000000000001")
	for _, body := range []string{"<svg onload='alert(1)'/>", "not an image", strings.Repeat("x", (5<<20)+1)} {
		if _, err := store.SaveProductImage(ctx, multipartFileHeader(t, "burger.png", body)); err == nil {
			t.Fatal("accepted invalid content")
		}
	}
	var huge bytes.Buffer
	if err := png.Encode(&huge, image.NewGray(image.Rect(0, 0, 4097, 1))); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveProductImage(ctx, multipartFileHeader(t, "huge.png", huge.String())); err == nil {
		t.Fatal("accepted oversized dimensions")
	}
	if calls != 0 {
		t.Fatal("invalid image reached storage")
	}
	var valid bytes.Buffer
	_ = png.Encode(&valid, image.NewGray(image.Rect(0, 0, 1, 1)))
	if _, err := store.SaveProductImage(context.Background(), multipartFileHeader(t, "ok.png", valid.String())); err == nil {
		t.Fatal("missing restaurant accepted")
	}
	if _, err := store.SaveProductImage(ctx, multipartFileHeader(t, "ok.png", valid.String())); err == nil || strings.Contains(err.Error(), "steal") {
		t.Fatal("redirect accepted or provider body leaked")
	}
	if calls != 1 {
		t.Fatal("unexpected storage calls")
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
