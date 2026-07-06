package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthProtectedRouteRejectsMissingToken(t *testing.T) {
	srv := NewServer("127.0.0.1:0", nil, Options{AuthToken: "secret-token"})
	handler, err := srv.handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthQueryTokenSetsCookieAndRedirects(t *testing.T) {
	srv := NewServer("127.0.0.1:0", nil, Options{AuthToken: "secret-token"})
	handler, err := srv.handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?token=secret-token", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/" {
		t.Fatalf("expected redirect to /, got %q", got)
	}

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != authCookieName || cookies[0].Value != "secret-token" {
		t.Fatalf("unexpected cookie: %#v", cookies[0])
	}
}

func TestAuthCookieAllowsProtectedRoute(t *testing.T) {
	srv := NewServer("127.0.0.1:0", nil, Options{AuthToken: "secret-token"})
	handler, err := srv.handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "secret-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "vminfo Dashboard") {
		t.Fatalf("expected dashboard HTML, got %q", rr.Body.String())
	}
}

func TestAuthQueryTokenAllowsStaticAssetWithoutRedirect(t *testing.T) {
	srv := NewServer("127.0.0.1:0", nil, Options{AuthToken: "secret-token"})
	handler, err := srv.handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/css/dashboard.css?token=secret-token", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "" {
		t.Fatalf("expected no redirect, got %q", got)
	}
}

func TestCheckWebSocketOriginWhenAuthEnabled(t *testing.T) {
	srv := NewServer("127.0.0.1:20021", nil, Options{AuthToken: "secret-token"})

	sameOriginReq := httptest.NewRequest(http.MethodGet, "/ws", nil)
	sameOriginReq.Host = "127.0.0.1:20021"
	sameOriginReq.Header.Set("Origin", "http://127.0.0.1:20021")
	if !srv.checkWebSocketOrigin(sameOriginReq) {
		t.Fatal("expected same-origin websocket request to pass")
	}

	crossOriginReq := httptest.NewRequest(http.MethodGet, "/ws", nil)
	crossOriginReq.Host = "127.0.0.1:20021"
	crossOriginReq.Header.Set("Origin", "http://evil.example")
	if srv.checkWebSocketOrigin(crossOriginReq) {
		t.Fatal("expected cross-origin websocket request to fail")
	}
}
