package auth

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

func (a *App) listSessions(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	rows, err := a.db.QueryContext(r.Context(), `SELECT id::text,created_at,expires_at,revoked_at FROM sessions WHERE user_id=$1 AND expires_at>now() ORDER BY created_at DESC LIMIT 100`, u.ID)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer rows.Close()
	type session struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
		Revoked   bool      `json:"revoked"`
		Current   bool      `json:"current"`
	}
	out := make([]session, 0)
	for rows.Next() {
		var s session
		var revoked sql.NullTime
		if rows.Scan(&s.ID, &s.CreatedAt, &s.ExpiresAt, &revoked) != nil {
			writeJSON(w, 500, message{"internal error"})
			return
		}
		s.Revoked = revoked.Valid
		s.Current = s.ID == r.Context().Value(sessionKey)
		out = append(out, s)
	}
	if rows.Err() != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) revokeSession(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	id := r.PathValue("session")
	res, err := a.db.ExecContext(r.Context(), `UPDATE sessions SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, id, u.ID)
	if err != nil {
		writeJSON(w, 400, message{"invalid session"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, message{"session not found"})
		return
	}
	a.audit(r.Context(), u.ID, u.ID, "", "session.revoked")
	if id == r.Context().Value(sessionKey) {
		clearCookie(w, a.secure)
	}
	writeJSON(w, 200, message{"session revoked"})
}

func (a *App) newSession(ctx context.Context, userID string) (string, error) {
	token, e := randomToken(32)
	if e != nil {
		return "", e
	}
	_, e = a.db.ExecContext(ctx, `INSERT INTO sessions(id,user_id,token_hash,expires_at) VALUES($1,$2,$3,$4)`, newID(), userID, a.hash(token), time.Now().Add(30*24*time.Hour))
	return token, e
}

func (a *App) signOut(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("session"); e == nil {
		_, _ = a.db.ExecContext(r.Context(), `UPDATE sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, a.hash(c.Value))
	}
	clearCookie(w, a.secure)
	a.audit(r.Context(), current(r).ID, current(r).ID, "", "session.revoked")
	writeJSON(w, 200, message{"signed out"})
}
