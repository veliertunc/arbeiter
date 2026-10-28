package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  USER@example.com ")
	if err != nil || got != "user@example.com" {
		t.Fatalf("normalizeEmail() = %q, %v", got, err)
	}
	for _, input := range []string{"", "not-an-email", "Name <user@example.com>", strings.Repeat("a", 250) + "@example.com"} {
		if _, err := normalizeEmail(input); err == nil {
			t.Errorf("normalizeEmail(%q) succeeded", input)
		}
	}
}

func TestValidSlug(t *testing.T) {
	for _, value := range []string{"acme", "team-42", "a-b-c"} {
		if !validSlug(value) {
			t.Errorf("validSlug(%q) = false", value)
		}
	}
	for _, value := range []string{"", "ab", "-abc", "abc-", "a--b", "UPPER", "has space"} {
		if validSlug(value) {
			t.Errorf("validSlug(%q) = true", value)
		}
	}
}

func TestRateClientIPTrustBoundary(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	r := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	r.RemoteAddr = "10.0.0.8:4321"
	r.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.4")
	if got := rateClientIP(r); got != "198.51.100.9" {
		t.Fatalf("trusted proxy client IP = %q", got)
	}
	r.RemoteAddr = "203.0.113.7:4321"
	if got := rateClientIP(r); got != "203.0.113.7" {
		t.Fatalf("untrusted peer client IP = %q", got)
	}
}

func TestAuthLifecyclePostgres(t *testing.T) {
	baseURL := os.Getenv("AUTH_TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("set AUTH_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", baseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "test_" + strings.ReplaceAll(newID(), "-", "")
	if _, err = admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, name := range []string{"001_init.sql", "002_operations.sql"} {
		contents, e := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if e != nil {
			t.Fatal(e)
		}
		for _, statement := range strings.Split(string(contents), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, e = db.ExecContext(ctx, statement); e != nil {
				t.Fatalf("migration %s: %v", name, e)
			}
		}
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("TOKEN_PEPPER", strings.Repeat("x", 48))
	t.Setenv("APP_ORIGIN", "http://localhost:3000")
	app, err := New(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler())
	defer server.Close()
	request := func(method, path, body string, cookie *http.Cookie) (*http.Response, map[string]any) {
		t.Helper()
		req, e := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		var payload map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &payload)
		}
		return res, payload
	}
	res, payload := request("POST", "/v1/sign-up", `{"email":"person@example.com","password":"this is a long password","name":"Person"}`, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("signup status %d: %v", res.StatusCode, payload)
	}
	ownerID, _ := payload["id"].(string)
	res, _ = request("POST", "/v1/sign-in", `{"email":"person@example.com","password":"this is a long password"}`, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("signin status %d", res.StatusCode)
	}
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatal("sign-in did not set a session cookie")
	}
	cookie := cookies[0]
	res, payload = request("GET", "/v1/me", "", cookie)
	if res.StatusCode != http.StatusOK || payload["email"] != "person@example.com" {
		t.Fatalf("me response: status=%d body=%v", res.StatusCode, payload)
	}
	res, _ = request("GET", "/v1/sessions", "", cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sessions response: status=%d", res.StatusCode)
	}
	res, payload = request("POST", "/v1/organizations", `{"name":"Test Org","slug":"test-org"}`, cookie)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("organization creation status %d: %v", res.StatusCode, payload)
	}
	orgID, _ := payload["id"].(string)
	res, payload = request("POST", "/v1/sign-up", `{"email":"member@example.com","password":"another long password"}`, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("member signup status %d: %v", res.StatusCode, payload)
	}
	memberID, _ := payload["id"].(string)
	res, _ = request("PUT", "/v1/organizations/"+orgID+"/members/"+memberID, `{"role":"member"}`, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("member add status %d", res.StatusCode)
	}
	res, _ = request("DELETE", "/v1/organizations/"+orgID+"/members/"+ownerID, "", cookie)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("owner membership was removable: %d", res.StatusCode)
	}
	res, _ = request("POST", "/v1/sign-out", "{}", cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sign out status %d", res.StatusCode)
	}
	res, _ = request("GET", "/v1/me", "", cookie)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked session accepted: %d", res.StatusCode)
	}
}
