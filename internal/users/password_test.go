package users

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCreationRejectsWeakOrTrimmedPasswords(t *testing.T) {
	svc := NewService(nil, nil)
	for _, password := range []string{"short", " a long passphrase", strings.Repeat("é", 37)} {
		_, err := svc.CreateUser(context.Background(), "First", "Last", "brand", "test@example.invalid", password, "admin")
		if !errors.Is(err, ErrInvalidUserData) {
			t.Fatalf("invalid password must be rejected before repository access: %v", err)
		}
	}
}
