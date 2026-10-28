package integration

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthenticationAPI(t *testing.T) {
	withTestApp(t, func(t *testing.T, _ *sql.DB, server *httptest.Server) {
		bad := call(t, server, http.MethodPost, "/v1/sign-up", `{"email":"not-an-email","password":"a sufficiently long password"}`)
		if bad.status != http.StatusBadRequest {
			t.Fatalf("invalid email accepted: %d", bad.status)
		}
		bad = call(t, server, http.MethodPost, "/v1/sign-up", `{"email":"person@example.test","password":"short"}`)
		if bad.status != http.StatusBadRequest {
			t.Fatalf("short password accepted: %d", bad.status)
		}
		res := call(t, server, http.MethodPost, "/v1/sign-up", `{"email":"  PERSON@EXAMPLE.TEST ","password":"a sufficiently long password","name":"Person"}`)
		if res.status != http.StatusCreated || object(t, res)["email"] != "person@example.test" {
			t.Fatalf("normalized sign-up failed: %d %s", res.status, res.body)
		}
		if res = call(t, server, http.MethodPost, "/v1/sign-up", `{"email":"person@example.test","password":"a sufficiently long password"}`); res.status != http.StatusConflict {
			t.Fatalf("duplicate sign-up status=%d", res.status)
		}
		if res = call(t, server, http.MethodPost, "/v1/sign-in", `{"email":"person@example.test","password":"wrong password"}`); res.status != http.StatusUnauthorized {
			t.Fatalf("bad credentials status=%d", res.status)
		}
		cookie := signIn(t, server, "person@example.test", "a sufficiently long password")
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("session cookie flags: %+v", cookie)
		}
		for label, body := range map[string]string{"malformed": "{", "unknown field": `{"unexpected":true}`, "multiple values": `{} {}`, "oversized": `{"name":"` + strings.Repeat("x", 1<<20) + `"}`} {
			res = call(t, server, http.MethodPatch, "/v1/me", body, cookie)
			if res.status != http.StatusBadRequest {
				t.Errorf("%s body status=%d", label, res.status)
			}
		}
	})
}
