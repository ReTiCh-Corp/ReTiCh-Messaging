package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/retich-corp/messaging/internal/response"
	"github.com/retich-corp/messaging/internal/service"
)

type SocialHandler struct {
	service service.SocialService
}

func NewSocialHandler(svc service.SocialService) *SocialHandler {
	return &SocialHandler{service: svc}
}

func (h *SocialHandler) RegisterRoutes(r *mux.Router) {
	// Reactions
	r.HandleFunc("/messages/{id}/reactions", h.AddReaction).Methods("POST")
	r.HandleFunc("/messages/{id}/reactions/{emoji}", h.RemoveReaction).Methods("DELETE")
	r.HandleFunc("/messages/{id}/reactions", h.ListReactions).Methods("GET")

	// Read receipts
	r.HandleFunc("/conversations/{id}/read", h.UpdateReadReceipt).Methods("POST")
	r.HandleFunc("/conversations/{id}/read", h.ListReadReceipts).Methods("GET")

	// Pinned messages
	r.HandleFunc("/conversations/{id}/pins", h.ListPinnedMessages).Methods("GET")
	r.HandleFunc("/conversations/{id}/pins", h.PinMessage).Methods("POST")
	r.HandleFunc("/conversations/{id}/pins/{messageId}", h.UnpinMessage).Methods("DELETE")
}

// --- Request types ---

type addReactionRequest struct {
	Emoji string `json:"emoji"`
}

func (req addReactionRequest) validate() map[string]string {
	errs := make(map[string]string)
	if req.Emoji == "" {
		errs["emoji"] = "emoji is required"
	} else if len(req.Emoji) > 50 {
		errs["emoji"] = "emoji must not exceed 50 characters"
	}
	return errs
}

type updateReadReceiptRequest struct {
	LastReadMessageID string `json:"last_read_message_id"`
}

func (req updateReadReceiptRequest) validate() (*uuid.UUID, map[string]string) {
	errs := make(map[string]string)
	if req.LastReadMessageID == "" {
		errs["last_read_message_id"] = "last_read_message_id is required"
		return nil, errs
	}
	parsed, err := uuid.Parse(req.LastReadMessageID)
	if err != nil {
		errs["last_read_message_id"] = "invalid UUID format"
		return nil, errs
	}
	return &parsed, errs
}

type pinMessageRequest struct {
	MessageID string `json:"message_id"`
}

func (req pinMessageRequest) validate() (*uuid.UUID, map[string]string) {
	errs := make(map[string]string)
	if req.MessageID == "" {
		errs["message_id"] = "message_id is required"
		return nil, errs
	}
	parsed, err := uuid.Parse(req.MessageID)
	if err != nil {
		errs["message_id"] = "invalid UUID format"
		return nil, errs
	}
	return &parsed, errs
}

// --- Handlers: Reactions ---

func (h *SocialHandler) AddReaction(w http.ResponseWriter, r *http.Request) {
	messageID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid message ID format, expected UUID")
		return
	}

	var req addReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	if errs := req.validate(); len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	reaction, err := h.service.AddReaction(r.Context(), service.AddReactionInput{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     req.Emoji,
	})
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			response.NotFound(w, "Message")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		if errors.Is(err, service.ErrReactionAlreadyExists) {
			response.Conflict(w, "Reaction already exists", nil)
			return
		}
		log.Printf("ERROR: add reaction: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusCreated, reaction)
}

func (h *SocialHandler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID, err := uuid.Parse(vars["id"])
	if err != nil {
		response.BadRequest(w, "Invalid message ID format, expected UUID")
		return
	}

	emoji := vars["emoji"]
	if emoji == "" {
		response.BadRequest(w, "Emoji is required")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	err = h.service.RemoveReaction(r.Context(), service.RemoveReactionInput{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	})
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			response.NotFound(w, "Message")
			return
		}
		log.Printf("ERROR: remove reaction: %v", err)
		response.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) ListReactions(w http.ResponseWriter, r *http.Request) {
	messageID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid message ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	reactions, err := h.service.ListReactions(r.Context(), messageID, userID)
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			response.NotFound(w, "Message")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: list reactions: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, reactions)
}

// --- Handlers: Read Receipts ---

func (h *SocialHandler) UpdateReadReceipt(w http.ResponseWriter, r *http.Request) {
	convID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	var req updateReadReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	msgID, errs := req.validate()
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	receipt, err := h.service.UpdateReadReceipt(r.Context(), service.UpdateReadReceiptInput{
		ConversationID:    convID,
		UserID:            userID,
		LastReadMessageID: *msgID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: update read receipt: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, receipt)
}

func (h *SocialHandler) ListReadReceipts(w http.ResponseWriter, r *http.Request) {
	convID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	receipts, err := h.service.ListReadReceipts(r.Context(), convID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: list read receipts: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, receipts)
}

// --- Handlers: Pinned Messages ---

func (h *SocialHandler) PinMessage(w http.ResponseWriter, r *http.Request) {
	convID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	var req pinMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	msgID, errs := req.validate()
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	pin, err := h.service.PinMessage(r.Context(), service.PinMessageInput{
		ConversationID: convID,
		MessageID:      *msgID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			response.NotFound(w, "Message")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.Forbidden(w, "Only owners and admins can pin messages")
			return
		}
		if errors.Is(err, service.ErrPinAlreadyExists) {
			response.Conflict(w, "Message is already pinned", nil)
			return
		}
		log.Printf("ERROR: pin message: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusCreated, pin)
}

func (h *SocialHandler) UnpinMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convID, err := uuid.Parse(vars["id"])
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	messageID, err := uuid.Parse(vars["messageId"])
	if err != nil {
		response.BadRequest(w, "Invalid message ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	err = h.service.UnpinMessage(r.Context(), service.UnpinMessageInput{
		ConversationID: convID,
		MessageID:      messageID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.Forbidden(w, "Only owners and admins can unpin messages")
			return
		}
		log.Printf("ERROR: unpin message: %v", err)
		response.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) ListPinnedMessages(w http.ResponseWriter, r *http.Request) {
	convID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	pins, err := h.service.ListPinnedMessages(r.Context(), convID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: list pinned messages: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, pins)
}
