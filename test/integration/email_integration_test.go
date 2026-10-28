package integration

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmailVerificationAndPasswordReset(t *testing.T) {
	withTestApp(t, func(t *testing.T, db *sql.DB, server *httptest.Server) {
		userID := signUp(t, server, "email@example.test", "original password is long")
		cookie := signIn(t, server, "email@example.test", "original password is long")
		if res := call(t, server, http.MethodPost, "/v1/email/verification", "{}", cookie); res.status != http.StatusServiceUnavailable {
			t.Fatalf("failed email delivery status=%d", res.status)
		}
		unknown := call(t, server, http.MethodPost, "/v1/password/reset", `{"email":"missing@example.test"}`)
		known := call(t, server, http.MethodPost, "/v1/password/reset", `{"email":"email@example.test"}`)
		if unknown.status != http.StatusAccepted || known.status != http.StatusAccepted {
			t.Fatalf("reset request enumeration response: unknown=%d known=%d", unknown.status, known.status)
		}
		verifiedToken := "verify-integration-token"
		insertActionToken(t, db, userID, verifiedToken, "verify", time.Now().Add(time.Hour))
		res := call(t, server, http.MethodPost, "/v1/email/verification/consume", `{"token":"`+verifiedToken+`"}`)
		if res.status != http.StatusOK {
			t.Fatalf("verification consume status=%d %s", res.status, res.body)
		}
		if res = call(t, server, http.MethodPost, "/v1/email/verification/consume", `{"token":"`+verifiedToken+`"}`); res.status != http.StatusBadRequest {
			t.Fatalf("verification token reused: %d", res.status)
		}
		var verified bool
		if err := db.QueryRow(`SELECT email_verified FROM users WHERE id=$1`, userID).Scan(&verified); err != nil {
			t.Fatal(err)
		}
		if !verified {
			t.Fatal("verification flag not persisted")
		}
		resetToken := "reset-integration-token"
		insertActionToken(t, db, userID, resetToken, "reset", time.Now().Add(time.Hour))
		res = call(t, server, http.MethodPost, "/v1/password/reset/consume", `{"token":"`+resetToken+`","password":"replacement password is long"}`)
		if res.status != http.StatusOK {
			t.Fatalf("reset consume status=%d %s", res.status, res.body)
		}
		if res = call(t, server, http.MethodGet, "/v1/me", "", cookie); res.status != http.StatusUnauthorized {
			t.Fatalf("password reset left session valid: %d", res.status)
		}
		if res = call(t, server, http.MethodPost, "/v1/sign-in", `{"email":"email@example.test","password":"original password is long"}`); res.status != http.StatusUnauthorized {
			t.Fatalf("old password still valid: %d", res.status)
		}
		_ = signIn(t, server, "email@example.test", "replacement password is long")
		expired := "expired-reset-token"
		insertActionToken(t, db, userID, expired, "reset", time.Now().Add(-time.Minute))
		if res = call(t, server, http.MethodPost, "/v1/password/reset/consume", `{"token":"`+expired+`","password":"another replacement password"}`); res.status != http.StatusBadRequest {
			t.Fatalf("expired reset token accepted: %d", res.status)
		}
	})
}
