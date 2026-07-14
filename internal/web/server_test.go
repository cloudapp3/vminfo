package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cloudapp3/vminfo/internal/collector"
)

func TestHandleProcessesRejectsNonGET(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/processes", nil)
	rr := httptest.NewRecorder()

	srv.handleProcesses(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestParseProcessQuery(t *testing.T) {
	tests := []struct {
		name    string
		values  url.Values
		want    processQueryOptions
		wantErr bool
	}{
		{
			name:   "defaults",
			values: url.Values{},
			want: processQueryOptions{
				sortKey: "cpu",
			},
		},
		{
			name: "filter sort and limit",
			values: url.Values{
				"filter": []string{"postgres"},
				"sort":   []string{"mem"},
				"limit":  []string{"10"},
			},
			want: processQueryOptions{
				filter:  "postgres",
				sortKey: "mem",
				limit:   10,
			},
		},
		{
			name: "q alias",
			values: url.Values{
				"q": []string{"ssh"},
			},
			want: processQueryOptions{
				filter:  "ssh",
				sortKey: "cpu",
			},
		},
		{
			name: "negative limit",
			values: url.Values{
				"limit": []string{"-1"},
			},
			wantErr: true,
		},
		{
			name: "non numeric limit",
			values: url.Values{
				"limit": []string{"many"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProcessQuery(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseProcessQuery returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("options = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestApplyProcessQueryFiltersSortsAndLimits(t *testing.T) {
	info := collector.ProcessInfo{
		Total: 4,
		List: []collector.ProcessEntry{
			{PID: 101, PPID: 1, Name: "nginx", Command: "nginx: worker", User: "www-data", CPUPercent: 12.5, MemPercent: 1.2, Status: "S"},
			{PID: 202, PPID: 1, Name: "postgres", Command: "postgres: writer", User: "postgres", CPUPercent: 3.1, MemPercent: 9.8, Status: "S"},
			{PID: 303, PPID: 202, Name: "postgres", Command: "postgres: checkpointer", User: "postgres", CPUPercent: 7.4, MemPercent: 5.6, Status: "R"},
			{PID: 404, PPID: 1, Name: "sleep", Command: "sleep 10", User: "root", CPUPercent: 0.0, MemPercent: 0.1, Status: "S"},
		},
	}

	got := applyProcessQuery(info, processQueryOptions{
		filter:  "postgres",
		sortKey: "mem",
		limit:   1,
	})

	if got.Total != info.Total {
		t.Fatalf("total = %d, want %d", got.Total, info.Total)
	}
	if len(got.List) != 1 {
		t.Fatalf("expected 1 process, got %d", len(got.List))
	}
	if got.List[0].PID != 202 {
		t.Fatalf("PID = %d, want 202", got.List[0].PID)
	}
}

func TestProcessEntryMatchesPIDPPIDCommandUserAndStatus(t *testing.T) {
	item := collector.ProcessEntry{
		PID:     303,
		PPID:    202,
		Name:    "postgres",
		Command: "postgres: checkpointer",
		User:    "postgres",
		Status:  "D",
	}

	for _, filter := range []string{"303", "202", "checkpointer", "postgres", "d"} {
		t.Run(filter, func(t *testing.T) {
			if !processEntryMatches(item, filter) {
				t.Fatalf("expected %q to match %+v", filter, item)
			}
		})
	}
}

func TestHandleNetDiagRejectsNonPOST(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/net/diag", nil)
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestHandleNetDiagRequiresTarget(t *testing.T) {
	srv := &Server{}
	req := newNetDiagRequest(`{"action":"dns"}`)
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing target, got %d", rr.Code)
	}
}

func TestHandleNetDiagUnknownAction(t *testing.T) {
	srv := &Server{}
	req := newNetDiagRequest(`{"action":"frob","target":"x"}`)
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 unknown action, got %d", rr.Code)
	}
}

func TestHandleNetDiagDNS(t *testing.T) {
	srv := &Server{}
	req := newNetDiagRequest(`{"action":"dns","target":"localhost"}`)
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleNetDiagPing(t *testing.T) {
	srv := &Server{}
	body := strings.NewReader(`{"action":"ping","target":"127.0.0.1","mode":"tcp","port":1,"count":1,"timeout_ms":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/net/diag", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.handleNetDiag(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleNetDiagRequiresJSONContentType(t *testing.T) {
	for _, contentType := range []string{"", "text/plain"} {
		t.Run(contentType, func(t *testing.T) {
			srv := &Server{}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/net/diag", strings.NewReader(`{"action":"dns","target":"localhost"}`))
			req.Header.Set("Content-Type", contentType)
			rr := httptest.NewRecorder()

			srv.handleNetDiag(rr, req)

			if rr.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
			}
		})
	}
}

func TestHandleNetDiagAcceptsJSONCharset(t *testing.T) {
	srv := &Server{}
	req := newNetDiagRequest(`{"action":"dns","target":"localhost"}`)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rr := httptest.NewRecorder()

	srv.handleNetDiag(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestHandleNetDiagRejectsCrossOrigin(t *testing.T) {
	srv := &Server{}
	req := newNetDiagRequest(`{"action":"dns","target":"localhost"}`)
	req.Host = "127.0.0.1:20021"
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()

	srv.handleNetDiag(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestNormalizeAndValidateNetDiagRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     NetDiagRequest
		want    NetDiagRequest
		wantErr bool
	}{
		{
			name: "ping defaults",
			req:  NetDiagRequest{Action: " PING ", Target: " 127.0.0.1 ", Port: 80},
			want: NetDiagRequest{Action: "ping", Target: "127.0.0.1", Port: 80, Count: 4, TimeoutMs: 2000, Mode: "tcp"},
		},
		{
			name:    "count too large",
			req:     NetDiagRequest{Action: "ping", Target: "127.0.0.1", Port: 80, Count: 11, TimeoutMs: 100, Mode: "tcp"},
			wantErr: true,
		},
		{
			name:    "timeout too large",
			req:     NetDiagRequest{Action: "port", Target: "127.0.0.1", Port: 80, TimeoutMs: 3001},
			wantErr: true,
		},
		{
			name:    "invalid port",
			req:     NetDiagRequest{Action: "port", Target: "127.0.0.1", Port: 65536, TimeoutMs: 100},
			wantErr: true,
		},
		{
			name:    "invalid mode",
			req:     NetDiagRequest{Action: "ping", Target: "127.0.0.1", Port: 80, Count: 1, TimeoutMs: 100, Mode: "udp"},
			wantErr: true,
		},
		{
			name: "icmp does not require port",
			req:  NetDiagRequest{Action: "ping", Target: "127.0.0.1", Count: 1, TimeoutMs: 100, Mode: "icmp"},
			want: NetDiagRequest{Action: "ping", Target: "127.0.0.1", Count: 1, TimeoutMs: 100, Mode: "icmp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizeNetDiagRequest(&tt.req)
			err := validateNetDiagRequest(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("validateNetDiagRequest returned error: %v", err)
			}
			if tt.req != tt.want {
				t.Fatalf("request = %+v, want %+v", tt.req, tt.want)
			}
		})
	}
}

func TestHandlerEnforcesSameOriginWithoutCORS(t *testing.T) {
	srv := NewServer("127.0.0.1:20021", collector.New(time.Second), Options{})
	handler, err := srv.handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	crossOrigin := httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil)
	crossOrigin.Host = "127.0.0.1:20021"
	crossOrigin.Header.Set("Origin", "http://evil.example")
	crossRecorder := httptest.NewRecorder()
	handler.ServeHTTP(crossRecorder, crossOrigin)
	if crossRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status = %d, want %d", crossRecorder.Code, http.StatusForbidden)
	}
	if got := crossRecorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin header %q", got)
	}

	sameOrigin := httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil)
	sameOrigin.Host = "127.0.0.1:20021"
	sameOrigin.Header.Set("Origin", "http://127.0.0.1:20021")
	sameRecorder := httptest.NewRecorder()
	handler.ServeHTTP(sameRecorder, sameOrigin)
	if sameRecorder.Code == http.StatusForbidden {
		t.Fatal("same-origin request was rejected")
	}

	nativeRequest := httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil)
	nativeRequest.Host = "127.0.0.1:20021"
	nativeRecorder := httptest.NewRecorder()
	handler.ServeHTTP(nativeRecorder, nativeRequest)
	if nativeRecorder.Code == http.StatusForbidden {
		t.Fatal("request without Origin was rejected")
	}
}

func TestHandlerRejectsDNSRebindingHostWithoutAuth(t *testing.T) {
	srv := NewServer("127.0.0.1:20021", collector.New(time.Second), Options{})
	handler, err := srv.handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system", nil)
	req.Host = "attacker.example:20021"
	req.Header.Set("Origin", "http://attacker.example:20021")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("DNS rebinding request status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestAllowedLoopbackHostValidatesHostAndPort(t *testing.T) {
	for _, host := range []string{"127.0.0.1:20021", "[::1]:20021", "localhost:20021", "localhost.:20021"} {
		if !isAllowedLoopbackHost(host, "127.0.0.1:20021") {
			t.Fatalf("expected loopback host %q to be allowed", host)
		}
	}
	for _, host := range []string{"attacker.example:20021", "127.0.0.1:8080", "", "0.0.0.0:20021"} {
		if isAllowedLoopbackHost(host, "127.0.0.1:20021") {
			t.Fatalf("expected host %q to be rejected", host)
		}
	}
}

func TestRequestHasSameOriginChecksSchemeAndDefaultPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Host = "example.test:80"
	req.Header.Set("Origin", "http://example.test")
	if !requestHasSameOrigin(req) {
		t.Fatal("expected equivalent HTTP default ports to match")
	}

	req.Header.Set("Origin", "https://example.test")
	if requestHasSameOrigin(req) {
		t.Fatal("expected cross-scheme origin to be rejected")
	}

	req.Header.Set("Origin", "null")
	if requestHasSameOrigin(req) {
		t.Fatal("expected opaque origin to be rejected")
	}

	req.Host = "example.test:443"
	req.Header.Set("Origin", "https://example.test")
	req.Header.Set("X-Forwarded-Proto", "https")
	if !requestHasSameOrigin(req) {
		t.Fatal("expected forwarded HTTPS origin to match")
	}
}

func TestNewHTTPServerTimeouts(t *testing.T) {
	srv := newHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 15*time.Second {
		t.Fatalf("ReadTimeout = %s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 15*time.Second {
		t.Fatalf("WriteTimeout = %s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s", srv.IdleTimeout)
	}
}

func TestShutdownBeforeStartPreventsFutureStartAndClosesHub(t *testing.T) {
	srv := NewServer("127.0.0.1:0", collector.New(time.Second), Options{})
	client := newWSClient(nil)
	if !srv.hub.tryRegister(client) {
		t.Fatal("expected registration to succeed")
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	assertClosed(t, client.done)
	if err := srv.Start(); !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Start error = %v, want http.ErrServerClosed", err)
	}
}

func TestStartAndShutdownAreSynchronized(t *testing.T) {
	srv := NewServer("127.0.0.1:0", collector.New(time.Second), Options{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		srv.lifecycleMu.Lock()
		started := srv.started
		srv.lifecycleMu.Unlock()
		if started {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not enter started state")
		}
		time.Sleep(time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Start returned %v, want http.ErrServerClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Shutdown")
	}
}

func TestWebSocketEnforcesOriginAndReadLimit(t *testing.T) {
	srv := NewServer("127.0.0.1:0", collector.New(time.Second), Options{})
	handler, err := srv.handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	defer srv.hub.closeAll()
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"

	crossHeaders := http.Header{"Origin": []string{"http://evil.example"}}
	crossConn, response, err := websocket.DefaultDialer.Dial(wsURL, crossHeaders)
	if crossConn != nil {
		_ = crossConn.Close()
	}
	if err == nil {
		t.Fatal("cross-origin WebSocket connection unexpectedly succeeded")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin response = %#v, want status %d", response, http.StatusForbidden)
	}

	sameHeaders := http.Header{"Origin": []string{httpServer.URL}}
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, sameHeaders)
	if err != nil {
		if response != nil {
			t.Fatalf("same-origin dial failed with status %d: %v", response.StatusCode, err)
		}
		t.Fatalf("same-origin dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, make([]byte, wsReadLimit+1)); err != nil {
		t.Fatalf("write oversized message: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("connection remained open after oversized inbound message")
	}
}

func newNetDiagRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/net/diag", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
