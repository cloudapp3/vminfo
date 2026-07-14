package web

import "testing"

func TestHubTryRegisterUnregisterIsIdempotent(t *testing.T) {
	hub := newHub(nil)
	client := newWSClient(nil)

	if !hub.tryRegister(client) {
		t.Fatal("expected first registration to succeed")
	}
	if !hub.tryRegister(client) {
		t.Fatal("expected duplicate registration to be idempotent")
	}
	if got := hub.clientCount(); got != 1 {
		t.Fatalf("expected 1 client after duplicate registration, got %d", got)
	}

	hub.unregister(client)
	hub.unregister(client)
	if got := hub.clientCount(); got != 0 {
		t.Fatalf("expected 0 clients after duplicate unregister, got %d", got)
	}
	assertClosed(t, client.done)
}

func TestHubRejectsClientsAboveCapacity(t *testing.T) {
	hub := newHub(nil)
	clients := make([]*wsClient, maxWSClients)
	for i := range clients {
		clients[i] = newWSClient(nil)
		if !hub.tryRegister(clients[i]) {
			t.Fatalf("registration %d unexpectedly failed", i)
		}
	}

	overflow := newWSClient(nil)
	if hub.tryRegister(overflow) {
		t.Fatalf("expected client %d to be rejected", maxWSClients+1)
	}
	if got := hub.clientCount(); got != maxWSClients {
		t.Fatalf("client count = %d, want %d", got, maxWSClients)
	}

	hub.closeAll()
}

func TestHubDropsOnlySlowClient(t *testing.T) {
	hub := newHub(nil)
	slow := newWSClient(nil)
	if !hub.tryRegister(slow) {
		t.Fatal("expected registration to succeed")
	}

	for i := 0; i < wsQueueSize; i++ {
		hub.broadcast([]byte("snapshot"))
	}
	if got := hub.clientCount(); got != 1 {
		t.Fatalf("client removed before queue filled: count = %d", got)
	}

	hub.broadcast([]byte("overflow"))
	if got := hub.clientCount(); got != 0 {
		t.Fatalf("slow client was not removed: count = %d", got)
	}
	assertClosed(t, slow.done)
}

func TestHubCloseAllRejectsFutureClients(t *testing.T) {
	hub := newHub(nil)
	first := newWSClient(nil)
	second := newWSClient(nil)
	if !hub.tryRegister(first) || !hub.tryRegister(second) {
		t.Fatal("expected initial registrations to succeed")
	}

	hub.closeAll()
	hub.closeAll()
	if got := hub.clientCount(); got != 0 {
		t.Fatalf("client count after closeAll = %d, want 0", got)
	}
	assertClosed(t, first.done)
	assertClosed(t, second.done)
	if hub.tryRegister(newWSClient(nil)) {
		t.Fatal("closed hub accepted a new client")
	}
}

func assertClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	default:
		t.Fatal("channel is not closed")
	}
}
