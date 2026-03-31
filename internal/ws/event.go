package ws

import "encoding/json"

// Event types sent over WebSocket.
const (
	EventMessageNew     = "message.new"
	EventMessageUpdated = "message.updated"
	EventMessageDeleted = "message.deleted"
	EventTypingStart    = "typing.start"
	EventTypingStop     = "typing.stop"
)

// Event is the envelope for all WebSocket messages.
type Event struct {
	Type           string          `json:"type"`
	ConversationID string          `json:"conversation_id"`
	Payload        json.RawMessage `json:"payload"`
}
