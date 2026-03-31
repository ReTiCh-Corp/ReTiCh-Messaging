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

type MessageHandler struct {
	service service.MessageService
}

func NewMessageHandler(svc service.MessageService) *MessageHandler {
	return &MessageHandler{service: svc}
}

func (h *MessageHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/conversations/{id}/messages", h.List).Methods("GET")
	r.HandleFunc("/conversations/{id}/messages", h.Create).Methods("POST")
	r.HandleFunc("/messages/{id}", h.GetByID).Methods("GET")
	r.HandleFunc("/messages/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/messages/{id}", h.Delete).Methods("DELETE")
}

// --- Request types ---

type createMessageRequest struct {
	Type      *string          `json:"type"`
	Content   *string          `json:"content"`
	Metadata  *json.RawMessage `json:"metadata"`
	ReplyToID *string          `json:"reply_to_id"`
}

func (req createMessageRequest) validate() (*uuid.UUID, map[string]string) {
	errs := make(map[string]string)

	validTypes := map[string]bool{"text": true, "image": true, "file": true, "audio": true, "video": true, "system": true}
	if req.Type != nil && !validTypes[*req.Type] {
		errs["type"] = "type must be one of: text, image, file, audio, video, system"
	}

	// text messages require content
	msgType := "text"
	if req.Type != nil {
		msgType = *req.Type
	}
	if msgType == "text" && (req.Content == nil || *req.Content == "") {
		errs["content"] = "content is required for text messages"
	}

	if req.Content != nil && len(*req.Content) > 10000 {
		errs["content"] = "content must not exceed 10000 characters"
	}

	var replyToID *uuid.UUID
	if req.ReplyToID != nil && *req.ReplyToID != "" {
		parsed, err := uuid.Parse(*req.ReplyToID)
		if err != nil {
			errs["reply_to_id"] = "invalid UUID format"
		} else {
			replyToID = &parsed
		}
	}

	return replyToID, errs
}

type updateMessageRequest struct {
	Content string `json:"content"`
}

func (req updateMessageRequest) validate() map[string]string {
	errs := make(map[string]string)
	if req.Content == "" {
		errs["content"] = "content is required"
	} else if len(req.Content) > 10000 {
		errs["content"] = "content must not exceed 10000 characters"
	}
	return errs
}

// --- Handlers ---

func (h *MessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	convID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	var req createMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	replyToID, errs := req.validate()
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	msgType := "text"
	if req.Type != nil {
		msgType = *req.Type
	}

	msg, err := h.service.Create(r.Context(), service.CreateMessageInput{
		ConversationID: convID,
		SenderID:       userID,
		Type:           msgType,
		Content:        req.Content,
		Metadata:       req.Metadata,
		ReplyToID:      replyToID,
	})
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.NotFound(w, "Conversation")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: create message: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusCreated, msg)
}

func (h *MessageHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid message ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	msg, err := h.service.GetByID(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			response.NotFound(w, "Message")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: get message %s: %v", id, err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, msg)
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
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

	limit, offset, errs := parsePagination(r)
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	result, err := h.service.ListByConversation(r.Context(), service.ListMessagesInput{
		ConversationID: convID,
		UserID:         userID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.NotFound(w, "Conversation")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: list messages for conversation %s: %v", convID, err)
		response.InternalError(w)
		return
	}

	response.SuccessWithPagination(w, result.Messages, response.PaginationMeta{
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *MessageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid message ID format, expected UUID")
		return
	}

	var req updateMessageRequest
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

	msg, err := h.service.Update(r.Context(), service.UpdateMessageInput{
		MessageID: id,
		UserID:    userID,
		Content:   req.Content,
	})
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			response.NotFound(w, "Message")
			return
		}
		if errors.Is(err, service.ErrNotMessageAuthor) {
			response.Forbidden(w, "Only the message author can edit this message")
			return
		}
		log.Printf("ERROR: update message %s: %v", id, err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, msg)
}

func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.BadRequest(w, "Invalid message ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	err = h.service.Delete(r.Context(), id, userID)
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
			response.Forbidden(w, "Insufficient permissions to delete this message")
			return
		}
		log.Printf("ERROR: delete message %s: %v", id, err)
		response.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
