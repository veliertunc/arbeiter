package auth

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type identity struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
	Verified bool   `json:"email_verified"`
}

func (a *App) me(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, current(r)) }

func (a *App) updateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	if utf8.RuneCountInString(in.Name) > 100 {
		writeJSON(w, 400, message{"name is too long"})
		return
	}
	u := current(r)
	_, e := a.db.ExecContext(r.Context(), `UPDATE users SET name=$1,updated_at=now() WHERE id=$2`, strings.TrimSpace(in.Name), u.ID)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	u.Name = strings.TrimSpace(in.Name)
	a.audit(r.Context(), u.ID, u.ID, "", "account.profile.updated")
	writeJSON(w, 200, u)
}

func (a *App) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	u := current(r)
	var hash string
	if a.db.QueryRowContext(r.Context(), `SELECT password_hash FROM users WHERE id=$1`, u.ID).Scan(&hash) != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeJSON(w, 401, message{"invalid password"})
		return
	}
	retiredPassword, err := randomToken(48)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	retiredHash, err := bcrypt.GenerateFromPassword([]byte(retiredPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	var owns bool
	if a.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM memberships WHERE user_id=$1 AND role='owner' AND org_id IN (SELECT id FROM organizations WHERE deleted_at IS NULL))`, u.ID).Scan(&owns) != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if owns {
		writeJSON(w, 409, message{"transfer organization ownership or delete owned organizations first"})
		return
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context(), `UPDATE users SET email='deleted+'||id::text||'@invalid.local',name='',password_hash=$1,disabled_at=now(),updated_at=now() WHERE id=$2`, string(retiredHash), u.ID); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, u.ID); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `DELETE FROM memberships WHERE user_id=$1`, u.ID); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE action_tokens SET consumed_at=now() WHERE user_id=$1 AND consumed_at IS NULL`, u.ID); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	a.audit(r.Context(), "", u.ID, "", "account.deleted")
	clearCookie(w, a.secure)
	w.WriteHeader(http.StatusNoContent)
}
