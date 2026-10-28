package auth

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) requestVerification(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if u.Verified {
		writeJSON(w, 200, message{"email already verified"})
		return
	}
	if e := a.issueActionToken(r.Context(), u.ID, "verify"); e != nil {
		a.log.Error("verification delivery failed", "user_id", u.ID, "error", e)
		writeJSON(w, 503, message{"email delivery unavailable"})
		return
	}
	writeJSON(w, 202, message{"verification instructions sent"})
}

func (a *App) consumeVerification(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !a.consumeActionToken(r.Context(), in.Token, "verify", "UPDATE users SET email_verified=true,updated_at=now() WHERE id=$1") {
		writeJSON(w, 400, message{"invalid or expired token"})
		return
	}
	var userID string
	if a.db.QueryRowContext(r.Context(), `SELECT user_id::text FROM action_tokens WHERE token_hash=$1 AND purpose='verify'`, a.hash(in.Token)).Scan(&userID) == nil {
		a.audit(r.Context(), userID, userID, "", "account.email.verified")
	}
	writeJSON(w, 200, message{"email verified"})
}

func (a *App) requestReset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	email, e := NormalizeEmail(in.Email)
	if e == nil {
		var id string
		if a.db.QueryRowContext(r.Context(), `SELECT id::text FROM users WHERE email=$1 AND disabled_at IS NULL`, email).Scan(&id) == nil {
			if e = a.issueActionToken(r.Context(), id, "reset"); e != nil {
				a.log.Error("reset delivery failed", "user_id", id, "error", e)
			}
		}
	}
	writeJSON(w, 202, message{"if the account exists, reset instructions will be sent"})
}

func (a *App) consumeReset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !ValidPassword(in.Password) {
		writeJSON(w, 400, message{"password must be 12–72 bytes"})
		return
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	tx, e := a.db.BeginTx(r.Context(), nil)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer tx.Rollback()
	var userID string
	e = tx.QueryRowContext(r.Context(), `UPDATE action_tokens SET consumed_at=now() WHERE token_hash=$1 AND purpose='reset' AND consumed_at IS NULL AND expires_at>now() RETURNING user_id::text`, a.hash(in.Token)).Scan(&userID)
	if e != nil {
		writeJSON(w, 400, message{"invalid or expired token"})
		return
	}
	if _, e = tx.ExecContext(r.Context(), `UPDATE users SET password_hash=$1,updated_at=now() WHERE id=$2`, string(hash), userID); e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	_, e = tx.ExecContext(r.Context(), `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if e = tx.Commit(); e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	a.audit(r.Context(), "", userID, "", "password.reset")
	writeJSON(w, 200, message{"password updated; please sign in again"})
}
