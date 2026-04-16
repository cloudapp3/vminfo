package web

import "testing"

func TestHubRegisterUnregisterIsIdempotent(t *testing.T) {
	hub := newHub(nil)
	client := &wsClient{}

	hub.register(client)
	hub.register(client)
	if got := hub.clientCount(); got != 1 {
		t.Fatalf("expected 1 client after duplicate register, got %d", got)
	}

	hub.unregister(client)
	hub.unregister(client)
	if got := hub.clientCount(); got != 0 {
		t.Fatalf("expected 0 clients after duplicate unregister, got %d", got)
	}
}
