package integration

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionLifecycle(t *testing.T) {
	withTestApp(t, func(t *testing.T, db *sql.DB, server *httptest.Server) {
		userID := signUp(t, server, "sessions@example.test", "a sufficiently long password")
		first := signIn(t, server, "sessions@example.test", "a sufficiently long password")
		second := signIn(t, server, "sessions@example.test", "a sufficiently long password")
		rows, err := db.Query(`SELECT token_hash FROM sessions WHERE user_id=$1`, userID)
		if err != nil {
			t.Fatal(err)
		}
		storedHashes := make([]string, 0, 2)
		for rows.Next() {
			var hash string
			if err = rows.Scan(&hash); err != nil {
				_ = rows.Close()
				t.Fatal(err)
			}
			storedHashes = append(storedHashes, hash)
		}
		if err = rows.Err(); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		_ = rows.Close()
		if len(storedHashes) != 2 {
			t.Fatalf("stored session hashes=%d", len(storedHashes))
		}
		for _, hash := range storedHashes {
			if len(hash) != 64 || hash == first.Value || hash == second.Value {
				t.Fatal("session token was not stored as a SHA-256 digest")
			}
		}
		res := call(t, server, http.MethodGet, "/v1/sessions", "", first)
		if res.status != http.StatusOK {
			t.Fatalf("list sessions status=%d", res.status)
		}
		var sessions []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		}
		if err := json.Unmarshal(res.body, &sessions); err != nil {
			t.Fatal(err)
		}
		if len(sessions) != 2 {
			t.Fatalf("sessions count=%d", len(sessions))
		}
		var other string
		for _, session := range sessions {
			if !session.Current {
				other = session.ID
			}
		}
		if other == "" {
			t.Fatal("session list did not identify current session")
		}
		res = call(t, server, http.MethodDelete, "/v1/sessions/"+other, "", first)
		if res.status != http.StatusOK {
			t.Fatalf("revoke session status=%d", res.status)
		}
		if res = call(t, server, http.MethodGet, "/v1/me", "", second); res.status != http.StatusUnauthorized {
			t.Fatalf("revoked session authorized: %d", res.status)
		}
		if res = call(t, server, http.MethodPost, "/v1/sign-out", "{}", first); res.status != http.StatusOK {
			t.Fatalf("sign-out status=%d", res.status)
		}
		if res = call(t, server, http.MethodGet, "/v1/me", "", first); res.status != http.StatusUnauthorized {
			t.Fatalf("signed-out session authorized: %d", res.status)
		}
	})
}
