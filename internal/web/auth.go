package web

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
)

const (
	authCookieName = "vminfo_web_token"
	authQueryKey   = "token"
)

type authConfig struct {
	token string
}

func newAuthConfig(token string) *authConfig {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return &authConfig{token: token}
}

func (a *authConfig) enabled() bool {
	return a != nil && a.token != ""
}

func (a *authConfig) wrap(next http.Handler) http.Handler {
	if !a.enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.hasValidCookie(r) {
			next.ServeHTTP(w, r)
			return
		}

		queryToken := strings.TrimSpace(r.URL.Query().Get(authQueryKey))
		if !a.matches(queryToken) {
			writeUnauthorized(w, r)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     authCookieName,
			Value:    queryToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   requestScheme(r) == "https",
			SameSite: http.SameSiteLaxMode,
		})

		if shouldRedirectAfterTokenAuth(r) {
			http.Redirect(w, r, stripTokenURL(r.URL), http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *authConfig) hasValidCookie(r *http.Request) bool {
	if !a.enabled() {
		return true
	}

	if cookie, err := r.Cookie(authCookieName); err == nil && a.matches(cookie.Value) {
		return true
	}
	return false
}

func (a *authConfig) matches(candidate string) bool {
	if !a.enabled() || candidate == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a.token), []byte(candidate)) == 1
}

func shouldRedirectAfterTokenAuth(r *http.Request) bool {
	if isWebSocketUpgrade(r) {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if r.URL.Path == "/" {
		return true
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html")
}

func stripTokenURL(u *neturl.URL) string {
	if u == nil {
		return "/"
	}
	clone := *u
	query := clone.Query()
	query.Del(authQueryKey)
	clone.RawQuery = query.Encode()
	if clone.Path == "" {
		clone.Path = "/"
	}
	return clone.RequestURI()
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"unauthorized"}`)
		return
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func wantsJSON(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "application/json")
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func splitHostPort(value string) (host, port string) {
	if strings.TrimSpace(value) == "" {
		return "", ""
	}

	host, port, err := net.SplitHostPort(value)
	if err == nil {
		return strings.ToLower(strings.TrimSpace(host)), strings.TrimSpace(port)
	}

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return strings.ToLower(strings.Trim(value, "[]")), ""
	}
	return strings.ToLower(strings.TrimSpace(value)), ""
}
