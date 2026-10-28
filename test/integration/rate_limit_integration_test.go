package integration

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticationRateLimit(t *testing.T) {
	withTestApp(t, func(t *testing.T, _ *sql.DB, server *httptest.Server) {
		for i := 0; i < 10; i++ {
			res := callFromIP(t, server, http.MethodPost, "/v1/sign-in", `{"email":"absent@example.test","password":"wrong"}`, "198.51.100.10")
			if res.status != http.StatusUnauthorized {
				t.Fatalf("attempt %d status=%d", i+1, res.status)
			}
		}
		otherClient := callFromIP(t, server, http.MethodPost, "/v1/sign-in", `{"email":"absent@example.test","password":"wrong"}`, "198.51.100.11")
		if otherClient.status != http.StatusUnauthorized {
			t.Fatalf("trusted client IPs shared a throttle bucket: %d", otherClient.status)
		}
		res := callFromIP(t, server, http.MethodPost, "/v1/sign-in", `{"email":"absent@example.test","password":"wrong"}`, "198.51.100.10")
		if res.status != http.StatusTooManyRequests || res.headers.Get("Retry-After") == "" {
			t.Fatalf("rate limit response: status=%d retry=%q", res.status, res.headers.Get("Retry-After"))
		}
	})
}
