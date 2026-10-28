package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type App struct {
	db       *sql.DB
	log      *slog.Logger
	secure   bool
	origin   string
	tokenKey []byte
}
type ctxKey string

const identityKey ctxKey = "identity"
const sessionKey ctxKey = "session_id"

type identity struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
	Verified bool   `json:"email_verified"`
}
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}
type orgInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}
type membershipInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}
type message struct {
	Message string `json:"message"`
}

func New(db *sql.DB, logger *slog.Logger) (*App, error) {
	secret := strings.TrimSpace(os.Getenv("TOKEN_PEPPER"))
	if len(secret) < 32 {
		return nil, fmt.Errorf("TOKEN_PEPPER must contain at least 32 characters")
	}
	if logger == nil {
		logger = slog.Default()
	}
	secure := strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
	origin := strings.TrimRight(os.Getenv("APP_ORIGIN"), "/")
	if os.Getenv("APP_ENV") != "development" {
		if !secure {
			return nil, fmt.Errorf("COOKIE_SECURE=true is required outside development")
		}
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("APP_ORIGIN must be an https origin")
		}
		for _, key := range []string{"SMTP_HOST", "SMTP_PORT", "SMTP_FROM"} {
			if strings.TrimSpace(os.Getenv(key)) == "" {
				return nil, fmt.Errorf("%s is required outside development", key)
			}
		}
		if _, err := mail.ParseAddress(os.Getenv("SMTP_FROM")); err != nil {
			return nil, fmt.Errorf("SMTP_FROM must be a valid email address")
		}
	}
	for _, raw := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(strings.TrimSpace(raw)); err != nil {
			return nil, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS entry")
		}
	}
	if port := strings.TrimSpace(os.Getenv("SMTP_PORT")); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return nil, fmt.Errorf("SMTP_PORT must be between 1 and 65535")
		}
	}
	return &App{db: db, log: logger, secure: secure, origin: origin, tokenKey: []byte(secret)}, nil
}
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, c := context.WithTimeout(r.Context(), 2*time.Second)
		defer c()
		if e := a.db.PingContext(ctx); e != nil {
			writeJSON(w, 503, message{"unavailable"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("POST /v1/sign-up", a.signUp)
	mux.HandleFunc("POST /v1/sign-in", a.signIn)
	mux.HandleFunc("POST /v1/sign-out", a.authenticated(a.signOut))
	mux.HandleFunc("GET /v1/sessions", a.authenticated(a.listSessions))
	mux.HandleFunc("DELETE /v1/sessions/{session}", a.authenticated(a.revokeSession))
	mux.HandleFunc("GET /v1/me", a.authenticated(a.me))
	mux.HandleFunc("PATCH /v1/me", a.authenticated(a.updateMe))
	mux.HandleFunc("DELETE /v1/me", a.authenticated(a.deleteAccount))
	mux.HandleFunc("POST /v1/email/verification", a.authenticated(a.requestVerification))
	mux.HandleFunc("POST /v1/email/verification/consume", a.consumeVerification)
	mux.HandleFunc("POST /v1/password/reset", a.requestReset)
	mux.HandleFunc("POST /v1/password/reset/consume", a.consumeReset)
	mux.HandleFunc("GET /v1/organizations", a.authenticated(a.listOrgs))
	mux.HandleFunc("POST /v1/organizations", a.authenticated(a.createOrg))
	mux.HandleFunc("DELETE /v1/organizations/{org}", a.authenticated(a.deleteOrg))
	mux.HandleFunc("GET /v1/organizations/{org}/members", a.authenticated(a.listMembers))
	mux.HandleFunc("PUT /v1/organizations/{org}/members/{user}", a.authenticated(a.putMember))
	mux.HandleFunc("DELETE /v1/organizations/{org}/members/{user}", a.authenticated(a.deleteMember))
	return a.middleware(mux)
}
func (a *App) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if a.origin != "" {
			if r.Header.Get("Origin") == a.origin {
				w.Header().Set("Access-Control-Allow-Origin", a.origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			} else if r.Header.Get("Origin") != "" {
				writeJSON(w, 403, message{"origin not allowed"})
				return
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		if r.URL.Path != "/health/live" && r.URL.Path != "/health/ready" {
			if limited, retry := a.rateLimited(r); limited {
				w.Header().Set("Retry-After", fmt.Sprint(retry))
				writeJSON(w, 429, message{"too many requests"})
				return
			}
			if r.Header.Get("Content-Type") != "" && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeJSON(w, 415, message{"application/json required"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (a *App) authenticated(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, e := r.Cookie("session")
		if e != nil {
			writeJSON(w, 401, message{"authentication required"})
			return
		}
		hash := a.hash(cookie.Value)
		var id, email, name, sessionID string
		var verified bool
		var expires time.Time
		e = a.db.QueryRowContext(r.Context(), `SELECT u.id::text,u.email,u.name,u.email_verified,s.expires_at,s.id::text FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND u.disabled_at IS NULL`, hash).Scan(&id, &email, &name, &verified, &expires, &sessionID)
		if e != nil || time.Now().After(expires) {
			clearCookie(w, a.secure)
			writeJSON(w, 401, message{"authentication required"})
			return
		}
		ctx := context.WithValue(r.Context(), identityKey, identity{id, email, name, verified})
		ctx = context.WithValue(ctx, sessionKey, sessionID)
		fn(w, r.WithContext(ctx))
	}
}
func current(r *http.Request) identity {
	v, _ := r.Context().Value(identityKey).(identity)
	return v
}

func (a *App) rateLimited(r *http.Request) (bool, int) {
	limit := 120
	scope := r.URL.Path
	switch r.URL.Path {
	case "/v1/sign-in":
		scope = "sign-in"
		limit = 10
	case "/v1/sign-up":
		scope = "sign-up"
		limit = 5
	case "/v1/password/reset":
		scope = "password-reset"
		limit = 5
	case "/v1/email/verification":
		scope = "verification"
		limit = 5
	case "/v1/password/reset/consume":
		scope = "password-reset-consume"
		limit = 10
	case "/v1/email/verification/consume":
		scope = "verification-consume"
		limit = 10
	default:
		if strings.HasPrefix(r.URL.Path, "/v1/organizations/") {
			scope = "organization-management"
		} else if strings.HasPrefix(r.URL.Path, "/v1/sessions/") {
			scope = "session-management"
		} else if r.URL.Path == "/v1/me" || r.URL.Path == "/v1/sessions" || r.URL.Path == "/v1/organizations" {
			scope = "authenticated-api"
		} else {
			return false, 0
		}
	}
	ip := rateClientIP(r)
	bucket := a.hash(ip + "|" + scope)
	var attempts int
	var start time.Time
	err := a.db.QueryRowContext(r.Context(), `INSERT INTO rate_limits(bucket,window_start,attempts) VALUES($1,now(),1) ON CONFLICT(bucket) DO UPDATE SET window_start=CASE WHEN rate_limits.window_start<=now()-interval '1 minute' THEN now() ELSE rate_limits.window_start END,attempts=CASE WHEN rate_limits.window_start<=now()-interval '1 minute' THEN 1 ELSE rate_limits.attempts+1 END RETURNING attempts,window_start`, bucket).Scan(&attempts, &start)
	if err != nil {
		a.log.Error("rate limit store unavailable", "error", err)
		return true, 60
	}
	if attempts <= limit {
		return false, 0
	}
	retry := int(time.Until(start.Add(time.Minute)).Seconds())
	if retry < 1 {
		retry = 1
	}
	return true, retry
}

func rateClientIP(r *http.Request) string {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	peerIP := net.ParseIP(peer)
	if peerIP == nil {
		return peer
	}
	var trusted []*net.IPNet
	for _, raw := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		_, subnet, e := net.ParseCIDR(strings.TrimSpace(raw))
		if e == nil {
			trusted = append(trusted, subnet)
		}
	}
	isTrusted := func(ip net.IP) bool {
		for _, subnet := range trusted {
			if subnet.Contains(ip) {
				return true
			}
		}
		return false
	}
	if !isTrusted(peerIP) {
		return peerIP.String()
	}
	forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	current := peerIP
	for i := len(forwarded) - 1; i >= 0; i-- {
		if !isTrusted(current) {
			break
		}
		candidate := net.ParseIP(strings.TrimSpace(forwarded[i]))
		if candidate == nil {
			return peerIP.String()
		}
		current = candidate
	}
	return current.String()
}

func (a *App) audit(ctx context.Context, actor, subject, org, event string) {
	_, err := a.db.ExecContext(ctx, `INSERT INTO audit_events(id,actor_user_id,subject_user_id,org_id,event_type) VALUES($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5)`, newID(), actor, subject, org, event)
	if err != nil {
		a.log.Error("audit event write failed", "event", event, "error", err)
	}
}

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

func (a *App) deleteOrg(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	org := r.PathValue("org")
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(r.Context(), `UPDATE organizations SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL AND EXISTS(SELECT 1 FROM memberships WHERE org_id=$1 AND user_id=$2 AND role='owner')`, org, u.ID)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, message{"organization not found"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `DELETE FROM memberships WHERE org_id=$1`, org); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	a.audit(r.Context(), u.ID, "", org, "organization.deleted")
	w.WriteHeader(http.StatusNoContent)
}
func (a *App) signUp(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	email, e := normalizeEmail(in.Email)
	if e != nil || !validPassword(in.Password) || utf8.RuneCountInString(in.Name) > 100 {
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
	email, e := normalizeEmail(in.Email)
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
	email, e := normalizeEmail(in.Email)
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
	if !validPassword(in.Password) {
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
func (a *App) issueActionToken(ctx context.Context, userID, purpose string) error {
	token, e := randomToken(32)
	if e != nil {
		return e
	}
	var email string
	if e = a.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id=$1`, userID).Scan(&email); e != nil {
		return e
	}
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `UPDATE action_tokens SET consumed_at=now() WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL`, userID, purpose)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO action_tokens(id,user_id,token_hash,purpose,expires_at) VALUES($1,$2,$3,$4,$5)`, newID(), userID, a.hash(token), purpose, time.Now().Add(30*time.Minute))
	if e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	if a.origin == "" {
		return fmt.Errorf("APP_ORIGIN is required for email actions")
	}
	link := a.origin + "/email/verification?token=" + token
	subject, body := "Verify your email", "Verify your email: "+link
	if purpose == "reset" {
		subject = "Reset your password"
		body = "Reset your password: " + a.origin + "/password/reset?token=" + token
	}
	return a.sendEmail(email, subject, body)
}
func (a *App) sendEmail(to, subject, body string) error {
	host, port, from := os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_FROM")
	if host == "" || port == "" || from == "" {
		return fmt.Errorf("SMTP_HOST, SMTP_PORT, and SMTP_FROM are required")
	}
	if _, e := mail.ParseAddress(from); e != nil {
		return fmt.Errorf("invalid SMTP_FROM")
	}
	addr := host + ":" + port
	var auth smtp.Auth
	if user := os.Getenv("SMTP_USERNAME"); user != "" {
		auth = smtp.PlainAuth("", user, os.Getenv("SMTP_PASSWORD"), host)
	}
	msg := []byte("From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n")
	conn, e := net.DialTimeout("tcp", addr, 10*time.Second)
	if e != nil {
		return e
	}
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	client, e := smtp.NewClient(conn, host)
	if e != nil {
		_ = conn.Close()
		return e
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}
	if e = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); e != nil {
		return e
	}
	if auth != nil {
		if e = client.Auth(auth); e != nil {
			return e
		}
	}
	if e = client.Mail(from); e != nil {
		return e
	}
	if e = client.Rcpt(to); e != nil {
		return e
	}
	w, e := client.Data()
	if e != nil {
		return e
	}
	if _, e = w.Write(msg); e != nil {
		_ = w.Close()
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	return client.Quit()
}
func (a *App) consumeActionToken(ctx context.Context, token, purpose, sqlUpdate string) bool {
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return false
	}
	defer tx.Rollback()
	var id string
	e = tx.QueryRowContext(ctx, `UPDATE action_tokens SET consumed_at=now() WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>now() RETURNING user_id::text`, a.hash(token), purpose).Scan(&id)
	if e != nil {
		return false
	}
	if _, e = tx.ExecContext(ctx, sqlUpdate, id); e != nil {
		return false
	}
	return tx.Commit() == nil
}

func (a *App) listOrgs(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	rows, e := a.db.QueryContext(r.Context(), `SELECT o.id::text,o.name,o.slug,m.role FROM organizations o JOIN memberships m ON m.org_id=o.id WHERE m.user_id=$1 AND o.deleted_at IS NULL ORDER BY o.created_at`, u.ID)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer rows.Close()
	out := make([]map[string]string, 0)
	for rows.Next() {
		var id, name, slug, role string
		if rows.Scan(&id, &name, &slug, &role) != nil {
			continue
		}
		out = append(out, map[string]string{"id": id, "name": name, "slug": slug, "role": role})
	}
	writeJSON(w, 200, out)
}
func (a *App) createOrg(w http.ResponseWriter, r *http.Request) {
	var in orgInput
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if len(in.Name) < 1 || len(in.Name) > 120 || !validSlug(in.Slug) {
		writeJSON(w, 400, message{"name and a valid slug (lowercase letters, numbers, hyphens) are required"})
		return
	}
	id := newID()
	tx, e := a.db.BeginTx(r.Context(), nil)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.Context(), `INSERT INTO organizations(id,name,slug) VALUES($1,$2,$3)`, id, in.Name, in.Slug); e != nil {
		writeJSON(w, 409, message{"organization could not be created"})
		return
	}
	if _, e = tx.ExecContext(r.Context(), `INSERT INTO memberships(org_id,user_id,role) VALUES($1,$2,'owner')`, id, current(r).ID); e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if e = tx.Commit(); e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	a.audit(r.Context(), current(r).ID, "", id, "organization.created")
	writeJSON(w, 201, map[string]string{"id": id, "name": in.Name, "slug": in.Slug, "role": "owner"})
}
func (a *App) listMembers(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	org := r.PathValue("org")
	if !a.isMember(r.Context(), org, u.ID) {
		writeJSON(w, 404, message{"organization not found"})
		return
	}
	rows, e := a.db.QueryContext(r.Context(), `SELECT u.id::text,u.email,u.name,m.role,u.email_verified FROM memberships m JOIN users u ON u.id=m.user_id WHERE m.org_id=$1 AND u.disabled_at IS NULL ORDER BY u.created_at`, org)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer rows.Close()
	type member struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Role     string `json:"role"`
		Verified bool   `json:"email_verified"`
	}
	out := make([]member, 0)
	for rows.Next() {
		var x member
		if rows.Scan(&x.ID, &x.Email, &x.Name, &x.Role, &x.Verified) == nil {
			out = append(out, x)
		}
	}
	writeJSON(w, 200, out)
}
func (a *App) putMember(w http.ResponseWriter, r *http.Request) {
	actor := current(r)
	org, target := r.PathValue("org"), r.PathValue("user")
	var in membershipInput
	if !decode(w, r, &in) {
		return
	}
	if in.UserID != "" && in.UserID != target {
		writeJSON(w, 400, message{"user_id does not match path"})
		return
	}
	if in.Role != "admin" && in.Role != "member" {
		writeJSON(w, 400, message{"role must be admin or member"})
		return
	}
	var role string
	e := a.db.QueryRowContext(r.Context(), `SELECT role FROM memberships WHERE org_id=$1 AND user_id=$2`, org, actor.ID).Scan(&role)
	if e != nil || role != "owner" && role != "admin" {
		writeJSON(w, 403, message{"organization admin access required"})
		return
	}
	if role != "owner" && in.Role != "member" {
		writeJSON(w, 403, message{"only owners can grant admin role"})
		return
	}
	res, e := a.db.ExecContext(r.Context(), `INSERT INTO memberships(org_id,user_id,role) SELECT $1,u.id,$3 FROM users u WHERE u.id=$2 AND u.disabled_at IS NULL ON CONFLICT(org_id,user_id) DO UPDATE SET role=EXCLUDED.role WHERE memberships.role<>'owner'`, org, target, in.Role)
	if e != nil {
		writeJSON(w, 404, message{"user or organization not found"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 409, message{"owner membership cannot be changed"})
		return
	}
	a.audit(r.Context(), actor.ID, target, org, "organization.member.upserted")
	writeJSON(w, 200, message{"member added or updated"})
}
func (a *App) deleteMember(w http.ResponseWriter, r *http.Request) {
	actor := current(r)
	org, target := r.PathValue("org"), r.PathValue("user")
	var role string
	e := a.db.QueryRowContext(r.Context(), `SELECT role FROM memberships WHERE org_id=$1 AND user_id=$2`, org, actor.ID).Scan(&role)
	if e != nil || role != "owner" && role != "admin" {
		writeJSON(w, 403, message{"organization admin access required"})
		return
	}
	if target == actor.ID {
		writeJSON(w, 400, message{"cannot remove yourself"})
		return
	}
	res, e := a.db.ExecContext(r.Context(), `DELETE FROM memberships WHERE org_id=$1 AND user_id=$2 AND role<>'owner'`, org, target)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, message{"member not found or cannot be removed"})
		return
	}
	a.audit(r.Context(), actor.ID, target, org, "organization.member.removed")
	writeJSON(w, 200, message{"member removed"})
}
func (a *App) isMember(ctx context.Context, org, user string) bool {
	var yes bool
	e := a.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM memberships WHERE org_id=$1 AND user_id=$2)`, org, user).Scan(&yes)
	return e == nil && yes
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		writeJSON(w, 400, message{"invalid JSON request"})
		return false
	}
	if e := d.Decode(&struct{}{}); !errors.Is(e, io.EOF) {
		writeJSON(w, 400, message{"request must contain one JSON object"})
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func normalizeEmail(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) > 254 {
		return "", errors.New("email too long")
	}
	a, e := mail.ParseAddress(v)
	if e != nil || a.Address != v || !strings.Contains(v, "@") {
		return "", errors.New("invalid email")
	}
	return v, nil
}
func validPassword(p string) bool { return len(p) >= 12 && len(p) <= 72 }
func validSlug(s string) bool {
	if len(s) < 3 || len(s) > 63 {
		return false
	}
	for i, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' && i > 0 && i < len(s)-1) || c == '-' && i > 0 && rune(s[i-1]) == '-' {
			return false
		}
	}
	return true
}
func newID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := fmt.Sprintf("%x", b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (a *App) hash(v string) string {
	h := sha256.New()
	h.Write(a.tokenKey)
	h.Write([]byte{0})
	h.Write([]byte(v))
	return fmt.Sprintf("%x", h.Sum(nil))
}
func setCookie(w http.ResponseWriter, v string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: v, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 60 * 60})
}
func clearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
