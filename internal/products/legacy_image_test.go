package products

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverLegacyImageRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	brand := "00000000-0000-4000-8000-000000000001"
	for _, raw := range []string{"/uploads/products/../../.env", "/uploads/products/00000000-0000-4000-8000-000000000001.svg", "/uploads/products/arbitrary.jpg", "https://example.invalid/image.jpg", "/uploads/products/../00000000-0000-4000-8000-000000000001.jpg", "/uploads/products/..\\00000000-0000-4000-8000-000000000001.jpg"} {
		if _, e := RecoverLegacyImage(context.Background(), nil, brand, root, raw); e == nil {
			t.Fatalf("accepted unsafe path %q", raw)
		}
	}
}
func TestRecoverLegacyImagePreservesBrandAndFile(t *testing.T) {
	root := t.TempDir()
	name := "00000000-0000-4000-8000-000000000002.jpg"
	original := []byte("existing image bytes; image decoder lives in the upload store")
	if e := os.WriteFile(filepath.Join(root, name), original, 0600); e != nil {
		t.Fatal(e)
	}
	called := false
	brand := "00000000-0000-4000-8000-000000000001"
	store := legacyImageStoreFunc(func(ctx context.Context, header *multipart.FileHeader) (string, error) {
		called = true
		if ctx.Value(imageBrandKey{}) != brand {
			t.Fatal("lost image brand scope")
		}
		f, e := header.Open()
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		raw, e := io.ReadAll(f)
		if e != nil || !bytes.Equal(raw, original) {
			t.Fatal("wrong recovery file")
		}
		return "https://example.invalid/recovered.jpg", nil
	})
	link, e := RecoverLegacyImage(context.Background(), store, brand, root, "/uploads/products/"+name)
	if e != nil || !called || link != "https://example.invalid/recovered.jpg" {
		t.Fatalf("recovery failed: %v", e)
	}
	after, e := os.ReadFile(filepath.Join(root, name))
	if e != nil || !bytes.Equal(after, original) {
		t.Fatal("original image changed")
	}

}

type legacyImageStoreFunc func(context.Context, *multipart.FileHeader) (string, error)

func (f legacyImageStoreFunc) SaveProductImage(ctx context.Context, h *multipart.FileHeader) (string, error) {
	return f(ctx, h)
}

func TestRecoverLegacyImageRejectsSymlinkOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.jpg")
	if e := os.WriteFile(outside, []byte("outside workspace"), 0600); e != nil {
		t.Fatal(e)
	}
	name := "00000000-0000-4000-8000-000000000002.jpg"
	if e := os.Symlink(outside, filepath.Join(root, name)); e != nil {
		t.Skip("symlinks unavailable")
	}
	if _, e := RecoverLegacyImage(context.Background(), nil, "00000000-0000-4000-8000-000000000001", root, "/uploads/products/"+name); e == nil {
		t.Fatal("followed symlink outside image root")
	}
}
