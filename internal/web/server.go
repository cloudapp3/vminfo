package web

import (
	"cmp"
	"compress/gzip"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

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
	server    *http.Server
	auth      *authConfig
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

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: handler,
	}

	// Start WS broadcast loop
	go s.broadcastLoop(context.Background())

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
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

	// WebSocket
	protectedMux.HandleFunc("/ws", s.handleWebSocket)

	protectedHandler := http.Handler(protectedMux)
	if s.auth.enabled() {
		protectedHandler = s.auth.wrap(protectedHandler)
	}

	rootMux := http.NewServeMux()
	rootMux.HandleFunc("/healthz", s.handleHealthz)
	rootMux.Handle("/", protectedHandler)

	return withCORS(rootMux, !s.auth.enabled()), nil
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

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSONGzip(w, r, map[string]any{
		"status":     "ok",
		"ws_clients": s.hub.clientCount(),
	})
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
	s.hub.register(client)

	// Send current snapshot immediately
	if data := s.collector.LatestJSONWithProcesses(r.Context()); data != nil {
		if err := client.writeMessage(websocket.TextMessage, data); err != nil {
			s.hub.unregister(client)
			return
		}
	}

	// Read loop (handles close/ping)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.hub.unregister(client)
			break
		}
	}
}

func (s *Server) checkWebSocketOrigin(r *http.Request) bool {
	if !s.auth.enabled() {
		return true
	}

	originValue := strings.TrimSpace(r.Header.Get("Origin"))
	if originValue == "" {
		return true
	}

	originURL, err := url.Parse(originValue)
	if err != nil {
		return false
	}
	return sameOriginHost(r.Host, originURL.Host)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
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

func withCORS(next http.Handler, enabled bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if enabled {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if enabled && r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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
