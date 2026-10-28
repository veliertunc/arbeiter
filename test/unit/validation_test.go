package unit_test

import (
	"strings"
	"testing"

	authsvc "github.com/arbeiter/auth-platform/internal/auth"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := authsvc.NormalizeEmail("  USER@example.test ")
	if err != nil || got != "user@example.test" {
		t.Fatalf("NormalizeEmail() = %q, %v", got, err)
	}
	for _, input := range []string{"", "not-an-email", "Name <user@example.test>", strings.Repeat("a", 250) + "@example.test"} {
		if _, err := authsvc.NormalizeEmail(input); err == nil {
			t.Errorf("NormalizeEmail(%q) succeeded", input)
		}
	}
}

func TestValidPassword(t *testing.T) {
	for _, password := range []string{strings.Repeat("a", 12), strings.Repeat("a", 72)} {
		if !authsvc.ValidPassword(password) {
			t.Errorf("ValidPassword(%d bytes) = false", len(password))
		}
	}
	for _, password := range []string{"short", strings.Repeat("a", 73)} {
		if authsvc.ValidPassword(password) {
			t.Errorf("ValidPassword(%d bytes) = true", len(password))
		}
	}
}

func TestValidOrganizationSlug(t *testing.T) {
	for _, slug := range []string{"acme", "team-42", "a-b-c"} {
		if !authsvc.ValidOrganizationSlug(slug) {
			t.Errorf("ValidOrganizationSlug(%q) = false", slug)
		}
	}
	for _, slug := range []string{"", "ab", "-abc", "abc-", "a--b", "UPPER", "has space"} {
		if authsvc.ValidOrganizationSlug(slug) {
			t.Errorf("ValidOrganizationSlug(%q) = true", slug)
		}
	}
}
