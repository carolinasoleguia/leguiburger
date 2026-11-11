package products

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"leguiburger/internal/db"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
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
			Journal:    db.DB,
		}
	}

	return MisconfiguredProductImageStore{}
}

type SupabaseProductImageStore struct {
	Journal    *gorm.DB
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

	body, err := io.ReadAll(io.LimitReader(file, (5<<20)+1))
	if err != nil {
		return "", err
	}

	if len(body) > 5<<20 {
		return "", errors.New("La imagen supera los 5 MB")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || (format != "jpeg" && format != "png") {
		return "", errors.New("Usá una imagen JPEG o PNG válida")
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || int64(config.Width)*int64(config.Height) > 8000000 {
		return "", errors.New("La imagen supera las dimensiones permitidas (4096 px, 8 megapíxeles)")
	}
	decoded, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return "", errors.New("La imagen está dañada")
	}
	var clean bytes.Buffer
	ext, mimeType := ".png", "image/png"
	if format == "jpeg" {
		ext, mimeType = ".jpg", "image/jpeg"
		err = jpeg.Encode(&clean, decoded, &jpeg.Options{Quality: 85})
	} else {
		err = png.Encode(&clean, decoded)
	}
	if err != nil {
		return "", err
	}
	body = clean.Bytes()
	brand, _ := ctx.Value(imageBrandKey{}).(string)
	if _, err := uuid.Parse(brand); err != nil {
		return "", errors.New("No se pudo verificar el restaurante de la imagen")
	}
	objectPath := path.Join("products", brand, uuid.NewString()+ext)
	publicURL, err := publicSupabaseObjectURL(s.BaseURL, s.Bucket, objectPath)
	if err != nil {
		return "", err
	}
	// The upload journal commits independently of the product transaction. A failed
	// save or an unknown network result can therefore be cleaned up later.
	if s.Journal != nil {
		if err = s.Journal.WithContext(ctx).Exec(`INSERT INTO legui_private.staged_product_images(object_path,public_url,brand_id) VALUES(?,?,?)`, objectPath, publicURL, brand).Error; err != nil {
			return "", err
		}
	}
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
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("x-upsert", "false")

	client := s.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	bounded := *client
	bounded.Timeout = 20 * time.Second
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := bounded.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("No se pudo guardar la imagen (estado %d)", resp.StatusCode)
	}

	return publicURL, nil
}

type imageBrandKey struct{}

func withImageBrand(ctx context.Context, brand string) context.Context {
	return context.WithValue(ctx, imageBrandKey{}, brand)
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
