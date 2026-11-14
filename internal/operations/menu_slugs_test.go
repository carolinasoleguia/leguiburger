package operations

import (
	"strings"
	"testing"
)

func TestPublicSlugValidation(t *testing.T) {
	for _, v := range []string{"", "UPPER", "with space", "../other", "a/b", "a%2fb", "a--b", "-start", "end-", strings.Repeat("a", 81)} {
		if publicSlug(v) {
			t.Fatalf("invalid public slug accepted: %q", v)
		}
	}
	for _, v := range []string{"poggs", "city-bell", "centro-2", strings.Repeat("a", 80)} {
		if !publicSlug(v) {
			t.Fatalf("valid slug rejected: %q", v)
		}
	}
}
