package products

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultProductImagesBucket = "catalog_images"

type ProductImageStore interface {
	SaveProductImage(ctx context.Context, fileHeader *multipart.FileHeader) (string, error)
}

func NewProductImageStoreFromEnv() ProductImageStore {
	supabaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_URL")), "/")
	supabaseKey := strings.TrimSpace(os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	if supabaseKey == "" {
		supabaseKey = strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY"))
	}
	bucket := strings.TrimSpace(os.Getenv("SUPABASE_STORAGE_BUCKET"))
	if bucket == "" {
		bucket = defaultProductImagesBucket
	}

	if supabaseURL != "" && supabaseKey != "" && !looksLikePlaceholder(supabaseKey) && !isS3StorageEndpoint(supabaseURL) {
		return &SupabaseProductImageStore{
			BaseURL:    supabaseURL,
			APIKey:     supabaseKey,
			Bucket:     bucket,
			HTTPClient: &http.Client{Timeout: 20 * time.Second},
		}
	}

	return MisconfiguredProductImageStore{}
}

type SupabaseProductImageStore struct {
	BaseURL    string
	APIKey     string
	Bucket     string
	HTTPClient *http.Client
}

func (s *SupabaseProductImageStore) SaveProductImage(ctx context.Context, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", nil
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	objectPath := path.Join("products", newProductImageFileName(fileHeader.Filename))
	uploadURL, err := url.JoinPath(s.BaseURL, "storage/v1/object", s.Bucket, objectPath)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("apikey", s.APIKey)
	req.Header.Set("Content-Type", contentTypeForFile(fileHeader))
	req.Header.Set("x-upsert", "true")

	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("supabase storage upload failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return publicSupabaseObjectURL(s.BaseURL, s.Bucket, objectPath)
}

func newProductImageFileName(originalName string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		ext = ".jpg"
	}
	return uuid.NewString() + ext
}

func contentTypeForFile(fileHeader *multipart.FileHeader) string {
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType != "" && contentType != "application/octet-stream" {
		return contentType
	}

	if byExtension := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileHeader.Filename))); byExtension != "" {
		return byExtension
	}

	return "application/octet-stream"
}

func publicSupabaseObjectURL(baseURL, bucket, objectPath string) (string, error) {
	return url.JoinPath(strings.TrimRight(baseURL, "/"), "storage/v1/object/public", bucket, objectPath)
}

type MisconfiguredProductImageStore struct{}

func (MisconfiguredProductImageStore) SaveProductImage(ctx context.Context, fileHeader *multipart.FileHeader) (string, error) {
	return "", errors.New("supabase storage is misconfigured")
}

func looksLikePlaceholder(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(normalized, "replace") ||
		strings.Contains(normalized, "your-") ||
		strings.Contains(normalized, "tu-") ||
		strings.Contains(normalized, "[") ||
		strings.Contains(normalized, "...")
}

func isS3StorageEndpoint(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(normalized, ".storage.supabase.co") ||
		strings.Contains(normalized, "/storage/v1/s3")
}
