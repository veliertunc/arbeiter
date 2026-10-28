package integration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	authsvc "github.com/arbeiter/auth-platform/internal/auth"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const testPepper = "integration-test-token-pepper-0123456789"

type apiResponse struct {
	status  int
	headers http.Header
	cookies []*http.Cookie
	body    []byte
}

func withTestApp(t *testing.T, run func(*testing.T, *sql.DB, *httptest.Server)) {
	t.Helper()
	baseURL := os.Getenv("AUTH_TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("set AUTH_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", baseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("test_%x", time.Now().UnixNano())
	if _, err = admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`) })
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
	t.Cleanup(func() { _ = db.Close() })
	for _, name := range []string{"001_init.sql", "002_operations.sql"} {
		contents, e := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if e != nil {
			t.Fatal(e)
		}
		tx, e := db.BeginTx(ctx, nil)
		if e != nil {
			t.Fatal(e)
		}
		for _, statement := range strings.Split(string(contents), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" || strings.EqualFold(statement, "BEGIN") || strings.EqualFold(statement, "COMMIT") {
				continue
			}
			if _, e = tx.ExecContext(ctx, statement); e != nil {
				_ = tx.Rollback()
				t.Fatalf("migration %s: %v", name, e)
			}
		}
		if e = tx.Commit(); e != nil {
			t.Fatalf("migration %s commit: %v", name, e)
		}
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("TOKEN_PEPPER", testPepper)
	t.Setenv("APP_ORIGIN", "http://localhost:3000")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.0/8,::1/128")
	t.Setenv("SMTP_HOST", "127.0.0.1")
	t.Setenv("SMTP_PORT", "1")
	t.Setenv("SMTP_FROM", "security@example.test")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	app, err := authsvc.New(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler())
	t.Cleanup(server.Close)
	run(t, db, server)
}

func call(t *testing.T, server *httptest.Server, method, path, body string, cookies ...*http.Cookie) apiResponse {
	return callFromIP(t, server, method, path, body, "", cookies...)
}

func callFromIP(t *testing.T, server *httptest.Server, method, path, body, forwardedIP string, cookies ...*http.Cookie) apiResponse {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if forwardedIP != "" {
		req.Header.Set("X-Forwarded-For", forwardedIP)
	}
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	content, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return apiResponse{status: res.StatusCode, headers: res.Header.Clone(), cookies: res.Cookies(), body: content}
}

func object(t *testing.T, res apiResponse) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(res.body, &value); err != nil {
		t.Fatalf("decode response %q: %v", res.body, err)
	}
	return value
}

func signUp(t *testing.T, server *httptest.Server, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	res := call(t, server, http.MethodPost, "/v1/sign-up", string(body))
	if res.status != http.StatusCreated {
		t.Fatalf("sign-up %s: status=%d body=%s", email, res.status, res.body)
	}
	return object(t, res)["id"].(string)
}

func signIn(t *testing.T, server *httptest.Server, email, password string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	res := call(t, server, http.MethodPost, "/v1/sign-in", string(body))
	if res.status != http.StatusOK {
		t.Fatalf("sign-in %s: status=%d body=%s", email, res.status, res.body)
	}
	for _, cookie := range res.cookies {
		if cookie.Name == "session" {
			return cookie
		}
	}
	t.Fatal("sign-in did not set session cookie")
	return nil
}

func insertActionToken(t *testing.T, db *sql.DB, userID, token, purpose string, expires time.Time) {
	t.Helper()
	digest := sha256.New()
	digest.Write([]byte(testPepper))
	digest.Write([]byte{0})
	digest.Write([]byte(token))
	tokenHash := hex.EncodeToString(digest.Sum(nil))
	_, err := db.Exec(`INSERT INTO action_tokens(id,user_id,token_hash,purpose,expires_at) VALUES($1,$2,$3,$4,$5)`, fmt.Sprintf("00000000-0000-4000-8000-%012d", time.Now().UnixNano()%1_000_000_000_000), userID, tokenHash, purpose, expires)
	if err != nil {
		t.Fatal(err)
	}
}
