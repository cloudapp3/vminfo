package web

import (
	"sync"

	"github.com/gorilla/websocket"
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
	c.conn.Close()
}

// WSHub manages WebSocket client connections.
type WSHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
}

func newHub() *WSHub {
	return &WSHub{
		clients: make(map[*wsClient]bool),
	}
}

func (h *WSHub) register(client *wsClient) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
}

func (h *WSHub) unregister(client *wsClient) {
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
	client.close()
}

func (h *WSHub) broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if err := client.writeMessage(websocket.TextMessage, data); err != nil {
			go h.unregister(client)
		}
	}
}

func (h *WSHub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
