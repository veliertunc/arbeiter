package integration

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserCRUDAndDeletion(t *testing.T) {
	withTestApp(t, func(t *testing.T, db *sql.DB, server *httptest.Server) {
		userID := signUp(t, server, "user@example.test", "a sufficiently long password")
		cookie := signIn(t, server, "user@example.test", "a sufficiently long password")
		if res := call(t, server, http.MethodGet, "/v1/me", "", cookie); res.status != http.StatusOK || object(t, res)["id"] != userID {
			t.Fatalf("get user failed: %d %s", res.status, res.body)
		}
		if res := call(t, server, http.MethodPatch, "/v1/me", `{"name":"Updated Name"}`, cookie); res.status != http.StatusOK || object(t, res)["name"] != "Updated Name" {
			t.Fatalf("update user failed: %d %s", res.status, res.body)
		}
		if res := call(t, server, http.MethodDelete, "/v1/me", `{"password":"incorrect password"}`, cookie); res.status != http.StatusUnauthorized {
			t.Fatalf("delete without password confirmation status=%d", res.status)
		}
		if res := call(t, server, http.MethodDelete, "/v1/me", `{"password":"a sufficiently long password"}`, cookie); res.status != http.StatusNoContent {
			t.Fatalf("delete account status=%d %s", res.status, res.body)
		}
		if res := call(t, server, http.MethodGet, "/v1/me", "", cookie); res.status != http.StatusUnauthorized {
			t.Fatalf("deleted account session remained valid: %d", res.status)
		}
		var email, name string
		var disabled sql.NullTime
		if err := db.QueryRow(`SELECT email,name,disabled_at FROM users WHERE id=$1`, userID).Scan(&email, &name, &disabled); err != nil {
			t.Fatal(err)
		}
		if !disabled.Valid || name != "" || strings.Contains(email, "user@example.test") {
			t.Fatalf("account PII was not scrubbed: %q %q disabled=%v", email, name, disabled.Valid)
		}
		var auditCount int
		if err := db.QueryRow(`SELECT count(*) FROM audit_events WHERE event_type='account.deleted' AND subject_user_id=$1`, userID).Scan(&auditCount); err != nil {
			t.Fatal(err)
		}
		if auditCount != 1 {
			t.Fatalf("account deletion audit count=%d", auditCount)
		}
	})
}
