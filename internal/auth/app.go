package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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
