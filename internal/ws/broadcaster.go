package ws

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
)

// HubBroadcaster implements service.EventBroadcaster using a WebSocket Hub.
type HubBroadcaster struct {
	hub *Hub
}

// NewHubBroadcaster creates a new HubBroadcaster.
func NewHubBroadcaster(hub *Hub) *HubBroadcaster {
	return &HubBroadcaster{hub: hub}
}

// BroadcastMessageEvent broadcasts a message event to all connected clients in a conversation.
func (b *HubBroadcaster) BroadcastMessageEvent(eventType string, conversationID uuid.UUID, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WS: failed to marshal payload: %v", err)
		return
	}

	event := Event{
		Type:           eventType,
		ConversationID: conversationID.String(),
		Payload:        payloadBytes,
	}

	b.hub.BroadcastToConversation(conversationID, event)
}
