package ws

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	db "github.com/retich-corp/messaging/internal/db"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Gateway handles CORS
	},
}

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub   *Hub
	store db.Store
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub, store db.Store) *Handler {
	return &Handler{hub: hub, store: store}
}

// ServeWS upgrades the HTTP connection to WebSocket.
// The user ID must be set in X-User-ID header by the gateway.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade failed for user %s: %v", userID, err)
		return
	}

	client := NewClient(h.hub, conn, userID)
	h.hub.Register(client)

	// Subscribe user to all their conversations
	h.subscribeToConversations(r.Context(), userID)

	go client.WritePump()
	go client.ReadPump()
}

// subscribeToConversations loads the user's conversations and subscribes them to each room.
func (h *Handler) subscribeToConversations(ctx context.Context, userID uuid.UUID) {
	conversations, err := h.store.ListConversationsByUser(ctx, db.ListConversationsByUserParams{
		UserID: userID,
		Limit:  500,
		Offset: 0,
	})
	if err != nil {
		log.Printf("WS: failed to load conversations for user %s: %v", userID, err)
		return
	}

	for _, conv := range conversations {
		h.hub.SubscribeUserSync(userID, conv.ID)
	}
	log.Printf("WS: user %s subscribed to %d conversations", userID, len(conversations))
}
