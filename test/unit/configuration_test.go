package unit_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	authsvc "github.com/arbeiter/auth-platform/internal/auth"
)

func setDevelopmentConfig(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("TOKEN_PEPPER", strings.Repeat("p", 48))
	t.Setenv("APP_ORIGIN", "http://localhost:3000")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")
	t.Setenv("SMTP_PORT", "")
}

func TestNewDevelopmentConfiguration(t *testing.T) {
	setDevelopmentConfig(t)
	if _, err := authsvc.New(nil, nil); err != nil {
		t.Fatalf("development config rejected: %v", err)
	}
}

func TestNewProductionConfigurationValidation(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("TOKEN_PEPPER", strings.Repeat("p", 48))
	t.Setenv("COOKIE_SECURE", "false")
	if _, err := authsvc.New(nil, nil); err == nil {
		t.Fatal("production config without secure cookies accepted")
	}
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("APP_ORIGIN", "http://auth.example.test")
	if _, err := authsvc.New(nil, nil); err == nil {
		t.Fatal("HTTP production origin accepted")
	}
	t.Setenv("APP_ORIGIN", "https://auth.example.test/path")
	if _, err := authsvc.New(nil, nil); err == nil {
		t.Fatal("non-origin production URL accepted")
	}
	t.Setenv("APP_ORIGIN", "https://auth.example.test")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")
	t.Setenv("SMTP_HOST", "smtp.example.test")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_FROM", "security@example.test")
	if _, err := authsvc.New(nil, nil); err != nil {
		t.Fatalf("valid production configuration rejected: %v", err)
	}
}

func TestLivenessResponseAndSecurityHeaders(t *testing.T) {
	setDevelopmentConfig(t)
	app, err := authsvc.New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/health/live", nil))
	if response.Code != 200 {
		t.Fatalf("liveness status=%d", response.Code)
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected security headers were not set")
	}
}
