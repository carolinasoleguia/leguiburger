package products

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"leguiburger/internal/auth"
	"leguiburger/internal/models"
	"leguiburger/internal/testutil"
)

type mockService struct {
	createProductFunc func(ctx context.Context, brandID, name, description string, basePrice float64, imageURL string) (*models.Product, error)
	updateProductFunc func(ctx context.Context, brandID, id, name, description string, basePrice *float64, imageURL string, isActive *bool) (*models.Product, error)
}

func (m *mockService) CreateProduct(ctx context.Context, brandID, name, description string, basePrice float64, imageURL string) (*models.Product, error) {
	return m.createProductFunc(ctx, brandID, name, description, basePrice, imageURL)
}

func (m *mockService) GetProduct(ctx context.Context, brandID, id string) (*models.Product, error) {
	return nil, nil
}

func (m *mockService) ListProducts(ctx context.Context, brandID string) ([]models.Product, error) {
	return nil, nil
}

func (m *mockService) UpdateProduct(ctx context.Context, brandID, id, name, description string, basePrice *float64, imageURL string, isActive *bool) (*models.Product, error) {
	return m.updateProductFunc(ctx, brandID, id, name, description, basePrice, imageURL, isActive)
}

func (m *mockService) DeleteProduct(ctx context.Context, brandID, id string) error {
	return nil
}

func TestHandler_CreateProduct_Success(t *testing.T) {
	brandID := "brand-ok"
	mockService := &mockService{
		createProductFunc: func(ctx context.Context, receivedBrandID, name, description string, basePrice float64, imageURL string) (*models.Product, error) {
			if receivedBrandID != brandID {
				t.Fatalf("se esperaba brand %s, se obtuvo %s", brandID, receivedBrandID)
			}
			return &models.Product{
				ID:          "new-id",
				BrandID:     receivedBrandID,
				Name:        name,
				Description: description,
				BasePrice:   basePrice,
				ImageURL:    imageURL,
				IsActive:    true,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	req := testutil.JSONRequest(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":        "Doble Cheddar",
		"description": "Burger con doble cheddar",
		"base_price":  4500.00,
		"image_url":   "https://example.com/burger.jpg",
	})
	req = testutil.WithClaims(t, req, &auth.Claims{Role: "admin", BrandID: &brandID})

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.Product
	testutil.DecodeJSONBody(t, rr, &response)

	if response.BrandID != brandID || response.Name != "Doble Cheddar" || response.BasePrice != 4500 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateProduct_MissingBrand(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/products", map[string]interface{}{})
	req = testutil.WithClaims(t, req, &auth.Claims{Role: auth.RoleOwner})

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_CreateProduct_MultipartImageUpload(t *testing.T) {
	brandID := "brand-ok"
	mockService := &mockService{
		createProductFunc: func(ctx context.Context, receivedBrandID, name, description string, basePrice float64, imageURL string) (*models.Product, error) {
			if receivedBrandID != brandID {
				t.Fatalf("se esperaba brand %s, se obtuvo %s", brandID, receivedBrandID)
			}
			if imageURL == "" {
				t.Fatal("se esperaba una URL de imagen generada")
			}
			return &models.Product{ID: "new-id", BrandID: receivedBrandID, Name: name, Description: description, BasePrice: basePrice, ImageURL: imageURL, IsActive: true}, nil
		},
	}

	handler := NewHandler(mockService)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "Doble Cheddar")
	_ = writer.WriteField("description", "Burger con cheddar")
	_ = writer.WriteField("base_price", "4500")
	_ = writer.WriteField("brand_id", brandID)
	part, err := writer.CreateFormFile("image_file", "burger.jpg")
	if err != nil {
		t.Fatalf("no se pudo crear el archivo multipart: %v", err)
	}
	_, _ = part.Write([]byte("fake-image-bytes"))
	if err := writer.Close(); err != nil {
		t.Fatalf("no se pudo cerrar el writer multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/products", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = testutil.WithClaims(t, req, &auth.Claims{Role: "admin", BrandID: &brandID})

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)
}

func TestHandler_UpdateProduct_MultipartImageUpload(t *testing.T) {
	brandID := "brand-ok"
	productID := "product-1"
	newPrice := 5200.0
	newActive := false
	mockService := &mockService{
		updateProductFunc: func(ctx context.Context, receivedBrandID, receivedID, name, description string, basePrice *float64, imageURL string, isActive *bool) (*models.Product, error) {
			if receivedBrandID != brandID {
				t.Fatalf("se esperaba brand %s, se obtuvo %s", brandID, receivedBrandID)
			}
			if receivedID != productID {
				t.Fatalf("se esperaba product %s, se obtuvo %s", productID, receivedID)
			}
			if basePrice == nil || *basePrice != newPrice {
				t.Fatalf("se esperaba precio %v, se obtuvo %v", newPrice, basePrice)
			}
			if isActive == nil || *isActive != newActive {
				t.Fatalf("se esperaba isActive %v, se obtuvo %v", newActive, isActive)
			}
			if imageURL == "" {
				t.Fatal("se esperaba una URL de imagen generada")
			}
			return &models.Product{ID: receivedID, BrandID: receivedBrandID, Name: name, Description: description, BasePrice: *basePrice, ImageURL: imageURL, IsActive: *isActive}, nil
		},
	}

	handler := NewHandler(mockService)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "Doble Cheddar")
	_ = writer.WriteField("description", "Burger con cheddar")
	_ = writer.WriteField("base_price", "5200")
	_ = writer.WriteField("is_active", "false")
	_ = writer.WriteField("brand_id", brandID)
	part, err := writer.CreateFormFile("image_file", "burger.jpg")
	if err != nil {
		t.Fatalf("no se pudo crear el archivo multipart: %v", err)
	}
	_, _ = part.Write([]byte("fake-image-bytes"))
	if err := writer.Close(); err != nil {
		t.Fatalf("no se pudo cerrar el writer multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/products/"+productID, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = testutil.WithClaims(t, req, &auth.Claims{Role: "admin", BrandID: &brandID})

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusOK)
}
