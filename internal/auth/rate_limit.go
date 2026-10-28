package auth

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

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
