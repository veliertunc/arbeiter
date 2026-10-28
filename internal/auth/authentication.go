package auth

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

func (a *App) signUp(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	email, e := NormalizeEmail(in.Email)
	if e != nil || !ValidPassword(in.Password) || utf8.RuneCountInString(in.Name) > 100 {
		writeJSON(w, 400, message{"valid email, password (12–72 bytes), and name are required"})
		return
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	id := newID()
	_, e = a.db.ExecContext(r.Context(), `INSERT INTO users(id,email,password_hash,name) VALUES($1,$2,$3,$4)`, id, email, string(hash), strings.TrimSpace(in.Name))
	if e != nil {
		writeJSON(w, 409, message{"account could not be created"})
		return
	}
	a.audit(r.Context(), id, id, "", "account.created")
	a.log.Info("account created", "user_id", id)
	writeJSON(w, 201, identity{ID: id, Email: email, Name: strings.TrimSpace(in.Name), Verified: false})
}

func (a *App) signIn(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	email, e := NormalizeEmail(in.Email)
	if e != nil || len(in.Password) > 72 {
		writeJSON(w, 401, message{"invalid email or password"})
		return
	}
	var id, hash, name string
	var verified bool
	e = a.db.QueryRowContext(r.Context(), `SELECT id::text,password_hash,name,email_verified FROM users WHERE email=$1 AND disabled_at IS NULL`, email).Scan(&id, &hash, &name, &verified)
	if e != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeJSON(w, 401, message{"invalid email or password"})
		return
	}
	token, e := a.newSession(r.Context(), id)
	if e != nil {
		a.log.Error("create session", "error", e)
		writeJSON(w, 500, message{"internal error"})
		return
	}
	setCookie(w, token, a.secure)
	a.audit(r.Context(), id, id, "", "session.created")
	writeJSON(w, 200, identity{ID: id, Email: email, Name: name, Verified: verified})
}
