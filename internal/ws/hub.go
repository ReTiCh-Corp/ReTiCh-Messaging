package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
)

// clientMessage wraps a message from a client for hub processing.
type clientMessage struct {
	client *Client
	data   []byte
}

// Hub maintains the set of active clients and broadcasts events to them.
type Hub struct {
	// clients maps userID → set of Client connections (a user can have multiple tabs)
	clients map[uuid.UUID]map[*Client]bool

	// rooms maps conversationID → set of userIDs subscribed
	rooms map[uuid.UUID]map[uuid.UUID]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan clientMessage

	// subscribe adds a client to a conversation room
	subscribe chan subscription

	mu sync.RWMutex
}

type subscription struct {
	userID         uuid.UUID
	conversationID uuid.UUID
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		rooms:      make(map[uuid.UUID]map[uuid.UUID]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan clientMessage),
		subscribe:  make(chan subscription),
	}
}

// Run starts the hub's event loop. Must be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.mu.Unlock()
			log.Printf("WS: user %s connected (%d connections)", client.userID, len(h.clients[client.userID]))

		case client := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[client.userID]; ok {
				delete(conns, client)
				close(client.send)
				if len(conns) == 0 {
					delete(h.clients, client.userID)
				}
			}
			h.mu.Unlock()
			log.Printf("WS: user %s disconnected", client.userID)

		case sub := <-h.subscribe:
			h.mu.Lock()
			if h.rooms[sub.conversationID] == nil {
				h.rooms[sub.conversationID] = make(map[uuid.UUID]bool)
			}
			h.rooms[sub.conversationID][sub.userID] = true
			h.mu.Unlock()

		case msg := <-h.broadcast:
			// Parse the event to get the conversation ID and forward to room
			var evt Event
			if err := json.Unmarshal(msg.data, &evt); err != nil {
				log.Printf("WS: invalid event from client: %v", err)
				continue
			}
			convID, err := uuid.Parse(evt.ConversationID)
			if err != nil {
				continue
			}
			// Broadcast to all users in the room except the sender
			h.broadcastToRoom(convID, msg.client.userID, msg.data)
		}
	}
}

// SubscribeUser subscribes a user to a conversation room.
func (h *Hub) SubscribeUser(userID, conversationID uuid.UUID) {
	h.subscribe <- subscription{userID: userID, conversationID: conversationID}
}

// SubscribeUserSync subscribes a user to a conversation room synchronously (for init).
func (h *Hub) SubscribeUserSync(userID, conversationID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[conversationID] == nil {
		h.rooms[conversationID] = make(map[uuid.UUID]bool)
	}
	h.rooms[conversationID][userID] = true
}

// BroadcastToConversation sends an event to all connected users in a conversation.
func (h *Hub) BroadcastToConversation(conversationID uuid.UUID, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("WS: failed to marshal event: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs, ok := h.rooms[conversationID]
	if !ok {
		return
	}

	for userID := range userIDs {
		if clients, ok := h.clients[userID]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
					// Client send buffer full, skip
					log.Printf("WS: dropping message for user %s (buffer full)", userID)
				}
			}
		}
	}
}

// broadcastToRoom sends data to all users in a room except excludeUser.
func (h *Hub) broadcastToRoom(conversationID, excludeUser uuid.UUID, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs, ok := h.rooms[conversationID]
	if !ok {
		return
	}

	for userID := range userIDs {
		if userID == excludeUser {
			continue
		}
		if clients, ok := h.clients[userID]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
				}
			}
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}
