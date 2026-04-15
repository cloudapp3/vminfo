package web

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/cloudapp3/vminfo/internal/collector"
)

//go:embed static/*
var staticFS embed.FS

// Server is the HTTP server for the web dashboard.
type Server struct {
	addr      string
	collector *collector.Collector
	hub       *WSHub
	server    *http.Server
}

// NewServer creates a new web server listening on addr (e.g. "127.0.0.1:20021").
func NewServer(addr string, c *collector.Collector) *Server {
	return &Server{
		addr:      addr,
		collector: c,
		hub:       newHub(),
	}
}

// Start starts the HTTP server. Blocks until the server exits.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static files (embedded SPA)
	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	// REST API
	mux.HandleFunc("/api/v1/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/v1/cpu", s.handleCPU)
	mux.HandleFunc("/api/v1/memory", s.handleMemory)
	mux.HandleFunc("/api/v1/disk", s.handleDisk)
	mux.HandleFunc("/api/v1/network", s.handleNetwork)
	mux.HandleFunc("/api/v1/processes", s.handleProcesses)
	mux.HandleFunc("/api/v1/system", s.handleSystem)
	mux.HandleFunc("/healthz", s.handleHealthz)

	// WebSocket
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: withCORS(mux),
	}

	// Start WS broadcast loop
	go s.broadcastLoop()

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) broadcastLoop() {
	ch := s.collector.Subscribe("web-hub")
	defer s.collector.Unsubscribe("web-hub")

	for snap := range ch {
		data, err := json.Marshal(snap)
		if err != nil {
			continue
		}
		s.hub.broadcast(data)
	}
}

// --- REST Handlers ---

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snap)
}

func (s *Server) handleCPU(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snap.CPU)
}

func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snap.Memory)
}

func (s *Server) handleDisk(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snap.Disk)
}

func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snap.Network)
}

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snap.Processes)
}

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	if snap == nil {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snap.System)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":     "ok",
		"ws_clients": s.hub.clientCount(),
	})
}

// --- WebSocket Handler ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := newWSClient(conn)
	s.hub.register(client)

	// Send current snapshot immediately (through the mutex-protected path)
	if snap := s.collector.Latest(); snap != nil {
		data, _ := json.Marshal(snap)
		client.writeMessage(websocket.TextMessage, data)
	}

	// Read loop (handles close/ping)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.hub.unregister(client)
			break
		}
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
