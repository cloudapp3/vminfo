package web

import (
	"cmp"
	"compress/gzip"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/collector"
)

var gzipPool = sync.Pool{
	New: func() any {
		gz, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return gz
	},
}

//go:embed static/*
var staticFS embed.FS

// Options configures the web dashboard server.
type Options struct {
	AuthToken string
}

// Server is the HTTP server for the web dashboard.
type Server struct {
	addr      string
	collector *collector.Collector
	hub       *WSHub
	auth      *authConfig

	lifecycleMu     sync.Mutex
	server          *http.Server
	cancelBroadcast context.CancelFunc
	started         bool
	stopped         bool
}

// NewServer creates a new web server listening on addr (e.g. "127.0.0.1:20021").
func NewServer(addr string, c *collector.Collector, opts Options) *Server {
	return &Server{
		addr:      addr,
		collector: c,
		hub:       newHub(c),
		auth:      newAuthConfig(opts.AuthToken),
	}
}

// Start starts the HTTP server. Blocks until the server exits.
func (s *Server) Start() error {
	handler, err := s.handler()
	if err != nil {
		return err
	}

	broadcastCtx, cancelBroadcast := context.WithCancel(context.Background())
	httpServer := newHTTPServer(s.addr, handler)

	s.lifecycleMu.Lock()
	if s.stopped {
		s.lifecycleMu.Unlock()
		cancelBroadcast()
		return http.ErrServerClosed
	}
	if s.started {
		s.lifecycleMu.Unlock()
		cancelBroadcast()
		return fmt.Errorf("web server already started")
	}
	s.started = true
	s.server = httpServer
	s.cancelBroadcast = cancelBroadcast
	s.lifecycleMu.Unlock()

	defer func() {
		cancelBroadcast()
		s.hub.closeAll()
		s.lifecycleMu.Lock()
		s.stopped = true
		s.cancelBroadcast = nil
		s.lifecycleMu.Unlock()
	}()
	if s.collector != nil {
		go s.broadcastLoop(broadcastCtx)
	}

	return httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.lifecycleMu.Lock()
	s.stopped = true
	httpServer := s.server
	cancelBroadcast := s.cancelBroadcast
	s.lifecycleMu.Unlock()

	if cancelBroadcast != nil {
		cancelBroadcast()
	}
	if s.hub != nil {
		s.hub.closeAll()
	}
	if httpServer == nil {
		return nil
	}
	return httpServer.Shutdown(ctx)
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (s *Server) handler() (http.Handler, error) {
	protectedMux := http.NewServeMux()

	// Static files (embedded SPA)
	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	protectedMux.Handle("/", http.FileServer(http.FS(staticContent)))

	// REST API
	protectedMux.HandleFunc("/api/v1/snapshot", s.handleSnapshot)
	protectedMux.HandleFunc("/api/v1/cpu", s.handleCPU)
	protectedMux.HandleFunc("/api/v1/memory", s.handleMemory)
	protectedMux.HandleFunc("/api/v1/disk", s.handleDisk)
	protectedMux.HandleFunc("/api/v1/network", s.handleNetwork)
	protectedMux.HandleFunc("/api/v1/processes", s.handleProcesses)
	protectedMux.HandleFunc("/api/v1/system", s.handleSystem)
	protectedMux.HandleFunc("/api/v1/health", s.handleHealth)
	protectedMux.HandleFunc("/api/v1/net/diag", s.handleNetDiag)

	// WebSocket
	protectedMux.HandleFunc("/ws", s.handleWebSocket)

	protectedHandler := http.Handler(protectedMux)
	if s.auth.enabled() {
		protectedHandler = s.auth.wrap(protectedHandler)
	}

	handler := requireSameOrigin(protectedHandler)
	if !s.auth.enabled() {
		handler = requireLoopbackHost(s.addr, handler)
	}
	return handler, nil
}

func (s *Server) broadcastLoop(ctx context.Context) {
	ch := s.collector.Subscribe("web-hub")
	defer s.collector.Unsubscribe("web-hub")

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			if data := s.collector.LatestJSON(); data != nil {
				s.hub.broadcast(data)
			}
		}
	}
}

// --- REST Handlers ---

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := s.collector.LatestWithProcesses(r.Context())
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSONGzip(w, r, snap)
}

func (s *Server) handleCPU(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSONGzip(w, r, snap.CPU)
}

func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSONGzip(w, r, snap.Memory)
}

func (s *Server) handleDisk(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSONGzip(w, r, snap.Disk)
}

func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSONGzip(w, r, snap.Network)
}

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := s.collector.LatestWithProcesses(r.Context())
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	opts, err := parseProcessQuery(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONGzip(w, r, applyProcessQuery(snap.Processes, opts))
}

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSONGzip(w, r, snap.System)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.LatestWithProcesses(r.Context())
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSONGzip(w, r, snap.Health)
}

// NetDiagRequest is the JSON body for POST /api/v1/net/diag.
type NetDiagRequest struct {
	Action    string `json:"action"`               // "dns" | "port" | "ping"
	Target    string `json:"target"`               // domain (dns) or host (port/ping)
	Port      int    `json:"port,omitempty"`       // port/ping action
	Server    string `json:"server,omitempty"`     // optional DNS server (dns action)
	TimeoutMs int    `json:"timeout_ms,omitempty"` // per-probe timeout (port/ping)
	Count     int    `json:"count,omitempty"`      // ping probe count
	Mode      string `json:"mode,omitempty"`       // ping mode: tcp | icmp
}

// handleNetDiag runs a network diagnostic on demand. Same-origin + protectedMux
// means it inherits token auth like the GET /api/v1/* routes.
func (s *Server) handleNetDiag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requestHasSameOrigin(r) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	var req NetDiagRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	normalizeNetDiagRequest(&req)
	if err := validateNetDiagRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var result any
	switch req.Action {
	case "dns":
		result = vminfo.ResolveDNS(ctx, req.Target, req.Server)
	case "port":
		result = vminfo.CheckPort(ctx, req.Target, req.Port, time.Duration(req.TimeoutMs)*time.Millisecond)
	case "ping":
		result = vminfo.Ping(ctx, req.Target, vminfo.PingOptions{
			Mode:    req.Mode,
			Count:   req.Count,
			Port:    req.Port,
			Timeout: time.Duration(req.TimeoutMs) * time.Millisecond,
		})
	case "ip":
		result = vminfo.LookupIP(ctx, req.Target, req.Server)
	default:
		http.Error(w, "unknown network diagnostic action", http.StatusBadRequest)
		return
	}
	writeJSONGzip(w, r, result)
}

func normalizeNetDiagRequest(req *NetDiagRequest) {
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	req.Target = strings.TrimSpace(req.Target)
	req.Server = strings.TrimSpace(req.Server)
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))

	switch req.Action {
	case "port":
		if req.TimeoutMs == 0 {
			req.TimeoutMs = 2000
		}
	case "ping":
		if req.Count == 0 {
			req.Count = 4
		}
		if req.TimeoutMs == 0 {
			req.TimeoutMs = 2000
		}
		if req.Mode == "" {
			req.Mode = "tcp"
		}
	}
}

func validateNetDiagRequest(req NetDiagRequest) error {
	if req.Target == "" {
		return fmt.Errorf("target is required")
	}

	switch req.Action {
	case "dns", "ip":
		return nil
	case "port":
		if req.Port < 1 || req.Port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
		if req.TimeoutMs < 1 || req.TimeoutMs > 3000 {
			return fmt.Errorf("timeout_ms must be between 1 and 3000")
		}
		return nil
	case "ping":
		if req.Count < 1 || req.Count > 10 {
			return fmt.Errorf("count must be between 1 and 10")
		}
		if req.TimeoutMs < 1 || req.TimeoutMs > 3000 {
			return fmt.Errorf("timeout_ms must be between 1 and 3000")
		}
		if req.Mode != "tcp" && req.Mode != "icmp" {
			return fmt.Errorf("mode must be tcp or icmp")
		}
		if req.Mode == "tcp" && (req.Port < 1 || req.Port > 65535) {
			return fmt.Errorf("port must be between 1 and 65535 for tcp ping")
		}
		if req.Mode == "icmp" && (req.Port < 0 || req.Port > 65535) {
			return fmt.Errorf("port must be between 1 and 65535 when provided")
		}
		return nil
	default:
		return fmt.Errorf("unknown action (want: dns | port | ping | ip)")
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: s.checkWebSocketOrigin,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := newWSClient(conn)
	if !s.hub.tryRegister(client) {
		_ = client.writeControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "server busy"))
		client.close()
		return
	}

	// Send current snapshot immediately
	if data := s.collector.LatestJSONWithProcesses(r.Context()); data != nil {
		if !client.enqueue(data) {
			s.hub.unregister(client)
			return
		}
	}

	go client.writePump(s.hub)
	client.readPump(s.hub)
}

func (s *Server) checkWebSocketOrigin(r *http.Request) bool {
	return requestHasSameOrigin(r)
}

// writeJSONGzip writes JSON with optional gzip compression.
func writeJSONGzip(w http.ResponseWriter, r *http.Request, v any) {
	w.Header().Set("Content-Type", "application/json")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() {
			gz.Close()
			gzipPool.Put(gz)
		}()
		json.NewEncoder(gz).Encode(v)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func requireSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requestHasSameOrigin(r) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireLoopbackHost(listenAddr string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedLoopbackHost(r.Host, listenAddr) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedLoopbackHost(requestHost, listenAddr string) bool {
	_, listenPort, err := net.SplitHostPort(strings.TrimSpace(listenAddr))
	if err != nil {
		return false
	}
	host, port := splitHostPort(requestHost)
	if port != "" && listenPort != "" && listenPort != "0" && port != listenPort {
		return false
	}
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func requestHasSameOrigin(r *http.Request) bool {
	originValue := strings.TrimSpace(r.Header.Get("Origin"))
	if originValue == "" {
		return true
	}

	originURL, err := url.Parse(originValue)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" || originURL.User != nil ||
		originURL.RawQuery != "" || originURL.Fragment != "" || (originURL.Path != "" && originURL.Path != "/") {
		return false
	}
	expectedScheme := requestScheme(r)
	if !strings.EqualFold(originURL.Scheme, expectedScheme) {
		return false
	}
	return sameOriginHostWithScheme(r.Host, originURL.Host, expectedScheme)
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); strings.EqualFold(forwarded, "http") || strings.EqualFold(forwarded, "https") {
		return strings.ToLower(forwarded)
	}
	return "http"
}

func sameOriginHostWithScheme(requestHost, originHost, scheme string) bool {
	reqHost, reqPort := splitHostPort(requestHost)
	originHostOnly, originPort := splitHostPort(originHost)
	if !strings.EqualFold(reqHost, originHostOnly) {
		return false
	}
	defaultPort := "80"
	if strings.EqualFold(scheme, "https") {
		defaultPort = "443"
	}
	if reqPort == "" {
		reqPort = defaultPort
	}
	if originPort == "" {
		originPort = defaultPort
	}
	return reqPort == originPort
}

type processQueryOptions struct {
	filter  string
	sortKey string
	limit   int
}

func parseProcessQuery(values url.Values) (processQueryOptions, error) {
	opts := processQueryOptions{
		filter:  firstNonEmptyQuery(values, "filter", "q"),
		sortKey: strings.ToLower(strings.TrimSpace(values.Get("sort"))),
	}
	if opts.sortKey == "" {
		opts.sortKey = "cpu"
	}

	limitValue := strings.TrimSpace(values.Get("limit"))
	if limitValue == "" {
		return opts, nil
	}
	limit, err := strconv.Atoi(limitValue)
	if err != nil || limit < 0 {
		return processQueryOptions{}, fmt.Errorf("limit must be a non-negative integer")
	}
	opts.limit = limit
	return opts, nil
}

func applyProcessQuery(info collector.ProcessInfo, opts processQueryOptions) collector.ProcessInfo {
	items := make([]collector.ProcessEntry, 0, len(info.List))
	for _, item := range info.List {
		if processEntryMatches(item, opts.filter) {
			items = append(items, item)
		}
	}

	sortProcessEntries(items, opts.sortKey)
	if opts.limit > 0 && opts.limit < len(items) {
		items = items[:opts.limit]
	}

	return collector.ProcessInfo{
		Total: info.Total,
		List:  items,
	}
}

func sortProcessEntries(items []collector.ProcessEntry, sortKey string) {
	sortKey = strings.ToLower(strings.TrimSpace(sortKey))
	slices.SortFunc(items, func(a, b collector.ProcessEntry) int {
		switch sortKey {
		case "mem":
			if a.MemPercent != b.MemPercent {
				return cmp.Compare(b.MemPercent, a.MemPercent)
			}
		case "pid":
			if a.PID != b.PID {
				return cmp.Compare(a.PID, b.PID)
			}
		case "name":
			aName := strings.ToLower(strings.TrimSpace(a.Name))
			bName := strings.ToLower(strings.TrimSpace(b.Name))
			if aName != bName {
				return cmp.Compare(aName, bName)
			}
		default:
			if a.CPUPercent != b.CPUPercent {
				return cmp.Compare(b.CPUPercent, a.CPUPercent)
			}
		}
		return cmp.Compare(a.PID, b.PID)
	})
}

func processEntryMatches(item collector.ProcessEntry, filter string) bool {
	query := strings.ToLower(strings.TrimSpace(filter))
	if query == "" {
		return true
	}
	fields := []string{
		strconv.FormatInt(int64(item.PID), 10),
		strconv.FormatInt(int64(item.PPID), 10),
		item.Name,
		item.Command,
		item.User,
		item.Status,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func firstNonEmptyQuery(values url.Values, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values.Get(key)); value != "" {
			return value
		}
	}
	return ""
}
