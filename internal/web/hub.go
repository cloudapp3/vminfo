package web

import (
	"sync"

	"github.com/gorilla/websocket"

	"github.com/cloudapp3/vminfo/internal/collector"
)

// wsClient wraps a websocket connection with a write mutex.
type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func newWSClient(conn *websocket.Conn) *wsClient {
	return &wsClient{conn: conn}
}

func (c *wsClient) writeMessage(msgType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(msgType, data)
}

func (c *wsClient) close() {
	if c == nil || c.conn == nil {
		return
	}
	c.conn.Close()
}

// WSHub manages WebSocket client connections.
type WSHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
	col     *collector.Collector
}

func newHub(col *collector.Collector) *WSHub {
	return &WSHub{
		clients: make(map[*wsClient]bool),
		col:     col,
	}
}

func (h *WSHub) register(client *wsClient) {
	var added bool
	h.mu.Lock()
	if !h.clients[client] {
		h.clients[client] = true
		added = true
	}
	h.mu.Unlock()
	if added && h.col != nil {
		h.col.RequestProcesses()
	}
}

func (h *WSHub) unregister(client *wsClient) {
	var removed bool
	h.mu.Lock()
	if h.clients[client] {
		delete(h.clients, client)
		removed = true
	}
	h.mu.Unlock()
	if !removed {
		return
	}
	client.close()
	if h.col != nil {
		h.col.ReleaseProcesses()
	}
}

func (h *WSHub) broadcast(data []byte) {
	h.mu.RLock()
	clients := make([]*wsClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if err := client.writeMessage(websocket.TextMessage, data); err != nil {
			h.unregister(client)
		}
	}
}

func (h *WSHub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
