package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
)

// NormalizeEmail validates an email address and returns its trimmed lowercase form.
func NormalizeEmail(v string) (string, error) {
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

// ValidPassword reports whether a password fits the service's current length policy.
func ValidPassword(p string) bool { return len(p) >= 12 && len(p) <= 72 }

// ValidOrganizationSlug reports whether a slug matches the organization URL policy.
func ValidOrganizationSlug(s string) bool {
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
