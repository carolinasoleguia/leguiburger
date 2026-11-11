package employees

import (
	"errors"
	"strings"
	"testing"
)

func TestCreationRejectsWeakOrTrimmedPasswords(t *testing.T) {
	svc := NewService(nil, nil)
	for _, password := range []string{"short", " a long passphrase", strings.Repeat("é", 37)} {
		_, err := svc.CreateEmployee(ownerTestContext(), "tenant", "First", "Last", "test@example.invalid", password, "", "employee")
		if !errors.Is(err, ErrInvalidEmployeeData) {
			t.Fatalf("invalid password must be rejected before repository access: %v", err)
		}
	}
}
