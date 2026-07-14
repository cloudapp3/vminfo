package web

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cloudapp3/vminfo/internal/collector"
)

const (
	maxWSClients = 64
	wsQueueSize  = 8
	wsReadLimit  = 4 << 10
	wsWriteWait  = 5 * time.Second
	wsPingPeriod = 30 * time.Second
	wsPongWait   = 60 * time.Second
)

// wsClient owns one WebSocket connection. Only writePump writes data frames;
// readPump is the sole reader.
type wsClient struct {
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func newWSClient(conn *websocket.Conn) *wsClient {
	return &wsClient{
		conn: conn,
		send: make(chan []byte, wsQueueSize),
		done: make(chan struct{}),
	}
}

func (c *wsClient) enqueue(data []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}

	select {
	case c.send <- data:
		return true
	case <-c.done:
		return false
	default:
		return false
	}
}

func (c *wsClient) readPump(h *WSHub) {
	defer h.unregister(c)

	c.conn.SetReadLimit(wsReadLimit)
	_ = c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *wsClient) writePump(h *WSHub) {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		h.unregister(c)
	}()

	for {
		select {
		case data := <-c.send:
			if err := c.writeMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.writeControl(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *wsClient) writeMessage(messageType int, data []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait)); err != nil {
		return err
	}
	return c.conn.WriteMessage(messageType, data)
}

func (c *wsClient) writeControl(messageType int, data []byte) error {
	return c.conn.WriteControl(messageType, data, time.Now().Add(wsWriteWait))
}

func (c *wsClient) close() {
	if c == nil {
		return
	}
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

// WSHub manages WebSocket client connections.
type WSHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
	closed  bool
	col     *collector.Collector
}

func newHub(col *collector.Collector) *WSHub {
	return &WSHub{
		clients: make(map[*wsClient]struct{}),
		col:     col,
	}
}

// tryRegister adds a client unless the hub is closed or at capacity.
func (h *WSHub) tryRegister(client *wsClient) bool {
	if client == nil {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return false
	}
	if _, ok := h.clients[client]; ok {
		return true
	}
	if len(h.clients) >= maxWSClients {
		return false
	}

	h.clients[client] = struct{}{}
	if h.col != nil {
		h.col.RequestProcesses()
	}
	return true
}

func (h *WSHub) unregister(client *wsClient) {
	var removed bool
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		removed = true
		if h.col != nil {
			h.col.ReleaseProcesses()
		}
	}
	h.mu.Unlock()

	if removed {
		client.close()
	}
}

func (h *WSHub) broadcast(data []byte) {
	if len(data) == 0 {
		return
	}

	var slowClients []*wsClient
	h.mu.RLock()
	for client := range h.clients {
		if !client.enqueue(data) {
			slowClients = append(slowClients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range slowClients {
		h.unregister(client)
	}
}

// closeAll permanently closes the hub and every registered connection.
func (h *WSHub) closeAll() {
	var clients []*wsClient
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	for client := range h.clients {
		clients = append(clients, client)
		delete(h.clients, client)
		if h.col != nil {
			h.col.ReleaseProcesses()
		}
	}
	h.mu.Unlock()

	for _, client := range clients {
		client.close()
	}
}

func (h *WSHub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
