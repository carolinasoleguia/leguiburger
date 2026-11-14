package products

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// RecoverLegacyImage reuses the current image validation, metadata stripping and
// upload journal. It only reads UUID-named image files inside the explicit root.
func RecoverLegacyImage(ctx context.Context, store ProductImageStore, brandID, root, legacyURL string) (string, error) {
	if _, e := uuid.Parse(brandID); e != nil {
		return "", errors.New("invalid image brand")
	}
	const prefix = "/uploads/products/"
	if !strings.HasPrefix(legacyURL, prefix) {
		return "", errors.New("not a legacy product image")
	}
	name := strings.TrimPrefix(legacyURL, prefix)
	ext := strings.ToLower(filepath.Ext(name))
	if (ext != ".jpg" && ext != ".jpeg" && ext != ".png") || filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
		return "", errors.New("invalid legacy image path")
	}
	if _, e := uuid.Parse(strings.TrimSuffix(name, filepath.Ext(name))); e != nil {
		return "", errors.New("invalid legacy image name")
	}
	base, e := filepath.EvalSymlinks(root)
	if e != nil {
		return "", errors.New("legacy image root unavailable")
	}
	base, e = filepath.Abs(base)
	if e != nil {
		return "", e
	}
	resolved, e := filepath.EvalSymlinks(filepath.Join(base, name))
	if e != nil {
		return "", errors.New("legacy image file unavailable")
	}
	relative, e := filepath.Rel(base, resolved)
	if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errors.New("legacy image escaped root")
	}
	file, e := os.Open(resolved)
	if e != nil {
		return "", e
	}
	defer file.Close()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, e := writer.CreateFormFile("image", name)
	if e != nil {
		return "", e
	}
	n, e := io.Copy(part, io.LimitReader(file, (5<<20)+1))
	if e != nil {
		return "", e
	}
	if n > 5<<20 {
		return "", errors.New("legacy image exceeds 5 MB")
	}
	if e = writer.Close(); e != nil {
		return "", e
	}
	form, e := multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary()).ReadForm(6 << 20)
	if e != nil {
		return "", e
	}
	defer form.RemoveAll()
	return store.SaveProductImage(withImageBrand(ctx, brandID), form.File["image"][0])
}
