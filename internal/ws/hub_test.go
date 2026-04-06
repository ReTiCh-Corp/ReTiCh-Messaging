package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newTestClient creates a Client with a buffered send channel (no real WebSocket).
func newTestClient(hub *Hub, userID uuid.UUID) *Client {
	return &Client{
		hub:    hub,
		userID: userID,
		send:   make(chan []byte, 256),
	}
}

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("expected non-nil hub")
	}
	if h.clients == nil || h.rooms == nil {
		t.Error("expected maps to be initialized")
	}
}

func TestHub_RegisterUnregister(t *testing.T) {
	h := NewHub()
	go h.Run()

	userID := uuid.New()
	client := newTestClient(h, userID)

	h.register <- client
	time.Sleep(20 * time.Millisecond)

	h.mu.RLock()
	_, connected := h.clients[userID]
	h.mu.RUnlock()
	if !connected {
		t.Error("expected client to be registered")
	}

	h.unregister <- client
	time.Sleep(20 * time.Millisecond)

	h.mu.RLock()
	_, stillConnected := h.clients[userID]
	h.mu.RUnlock()
	if stillConnected {
		t.Error("expected client to be unregistered")
	}
}

func TestHub_SubscribeUserSync(t *testing.T) {
	h := NewHub()

	userID := uuid.New()
	convID := uuid.New()

	h.SubscribeUserSync(userID, convID)

	h.mu.RLock()
	members, ok := h.rooms[convID]
	h.mu.RUnlock()

	if !ok {
		t.Fatal("expected conversation room to exist")
	}
	if !members[userID] {
		t.Error("expected user to be in conversation room")
	}
}

func TestHub_BroadcastToConversation_NoClients(t *testing.T) {
	h := NewHub()

	convID := uuid.New()
	userID := uuid.New()
	h.SubscribeUserSync(userID, convID)

	// No clients registered — should not panic
	evt := Event{Type: EventMessageNew, ConversationID: convID.String()}
	h.BroadcastToConversation(convID, evt)
}

func TestHub_BroadcastToConversation_WithClient(t *testing.T) {
	h := NewHub()
	go h.Run()

	userID := uuid.New()
	convID := uuid.New()

	client := newTestClient(h, userID)
	h.register <- client
	time.Sleep(20 * time.Millisecond)

	h.SubscribeUserSync(userID, convID)

	payload, _ := json.Marshal(map[string]string{"text": "hello"})
	evt := Event{
		Type:           EventMessageNew,
		ConversationID: convID.String(),
		Payload:        payload,
	}
	h.BroadcastToConversation(convID, evt)

	select {
	case data := <-client.send:
		var received Event
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}
		if received.Type != EventMessageNew {
			t.Errorf("expected type %s, got %s", EventMessageNew, received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for broadcast message")
	}
}

func TestHub_BroadcastToConversation_EmptyRoom(t *testing.T) {
	h := NewHub()

	// Broadcast to a room with no subscribers — should not panic
	convID := uuid.New()
	evt := Event{Type: EventMessageNew, ConversationID: convID.String()}
	h.BroadcastToConversation(convID, evt)
}
