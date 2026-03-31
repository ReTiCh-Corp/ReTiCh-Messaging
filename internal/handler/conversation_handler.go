package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/retich-corp/messaging/internal/response"
	"github.com/retich-corp/messaging/internal/service"
)

type ConversationHandler struct {
	service service.ConversationService
}

func NewConversationHandler(svc service.ConversationService) *ConversationHandler {
	return &ConversationHandler{service: svc}
}

func (h *ConversationHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/conversations", h.List).Methods("GET")
	r.HandleFunc("/conversations", h.Create).Methods("POST")
	r.HandleFunc("/conversations/{id}", h.GetByID).Methods("GET")
	r.HandleFunc("/conversations/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/conversations/{id}", h.Archive).Methods("DELETE")
	r.HandleFunc("/conversations/{id}/participants", h.AddParticipants).Methods("POST")
	r.HandleFunc("/conversations/{id}/participants/{userId}", h.RemoveParticipant).Methods("DELETE")
}

// --- Request types ---

type createConversationRequest struct {
	Type           string   `json:"type"`
	Name           *string  `json:"name"`
	Description    *string  `json:"description"`
	AvatarURL      *string  `json:"avatar_url"`
	ParticipantIDs []string `json:"participant_ids"`
}

func (req createConversationRequest) validate(creatorID uuid.UUID) ([]uuid.UUID, map[string]string) {
	errs := make(map[string]string)

	validTypes := map[string]bool{"direct": true, "group": true, "channel": true}
	if req.Type == "" {
		errs["type"] = "type is required"
	} else if !validTypes[req.Type] {
		errs["type"] = "type must be one of: direct, group, channel"
	}

	if req.Type == "direct" {
		if req.Name != nil {
			errs["name"] = "direct conversations cannot have a name"
		}
		if req.Description != nil {
			errs["description"] = "direct conversations cannot have a description"
		}
	}

	if req.Type == "group" || req.Type == "channel" {
		if req.Name == nil || *req.Name == "" {
			errs["name"] = "name is required for group and channel conversations"
		}
	}

	if req.Name != nil && len(*req.Name) > 100 {
		errs["name"] = "name must not exceed 100 characters"
	}

	if req.AvatarURL != nil && len(*req.AvatarURL) > 500 {
		errs["avatar_url"] = "avatar_url must not exceed 500 characters"
	}

	if len(req.ParticipantIDs) == 0 {
		errs["participant_ids"] = "at least one participant is required"
	}

	// Parse and validate participant UUIDs
	var participantIDs []uuid.UUID
	for i, idStr := range req.ParticipantIDs {
		pid, err := uuid.Parse(idStr)
		if err != nil {
			errs["participant_ids"] = "invalid UUID at index " + strconv.Itoa(i)
			break
		}
		if pid == creatorID {
			errs["participant_ids"] = "participant_ids must not include the creator"
			break
		}
		participantIDs = append(participantIDs, pid)
	}

	if req.Type == "direct" && len(req.ParticipantIDs) != 1 && errs["participant_ids"] == "" {
		errs["participant_ids"] = "direct conversation requires exactly one participant"
	}

	return participantIDs, errs
}

type updateConversationRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	AvatarURL   *string `json:"avatar_url"`
}

func (req updateConversationRequest) validate() map[string]string {
	errs := make(map[string]string)

	if req.Name == nil && req.Description == nil && req.AvatarURL == nil {
		errs["body"] = "at least one field (name, description, avatar_url) must be provided"
	}

	if req.Name != nil {
		if *req.Name == "" {
			errs["name"] = "name cannot be empty"
		} else if len(*req.Name) > 100 {
			errs["name"] = "name must not exceed 100 characters"
		}
	}

	if req.AvatarURL != nil && len(*req.AvatarURL) > 500 {
		errs["avatar_url"] = "avatar_url must not exceed 500 characters"
	}

	return errs
}

type addParticipantsRequest struct {
	ParticipantIDs []string `json:"participant_ids"`
}

func (req addParticipantsRequest) validate() ([]uuid.UUID, map[string]string) {
	errs := make(map[string]string)

	if len(req.ParticipantIDs) == 0 {
		errs["participant_ids"] = "at least one participant is required"
		return nil, errs
	}

	if len(req.ParticipantIDs) > 50 {
		errs["participant_ids"] = "cannot add more than 50 participants at once"
		return nil, errs
	}

	var ids []uuid.UUID
	for i, idStr := range req.ParticipantIDs {
		pid, err := uuid.Parse(idStr)
		if err != nil {
			errs["participant_ids"] = "invalid UUID at index " + strconv.Itoa(i)
			return nil, errs
		}
		ids = append(ids, pid)
	}

	return ids, errs
}

// --- Handlers ---

func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	participantIDs, errs := req.validate(userID)
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	conv, err := h.service.Create(r.Context(), service.CreateConversationInput{
		Type:           req.Type,
		Name:           req.Name,
		Description:    req.Description,
		AvatarURL:      req.AvatarURL,
		CreatorID:      userID,
		ParticipantIDs: participantIDs,
	})
	if err != nil {
		var existingErr *service.ExistingConversationError
		if errors.As(err, &existingErr) {
			response.Conflict(w, "Direct conversation already exists between these users", existingErr.Conversation)
			return
		}
		if errors.Is(err, service.ErrDirectRequiresOneParticipant) {
			response.ValidationError(w, map[string]string{"participant_ids": "direct conversation requires exactly one participant"})
			return
		}
		if errors.Is(err, service.ErrInvalidUserIDs) {
			response.ValidationError(w, map[string]string{"participant_ids": "one or more user IDs do not exist"})
			return
		}
		log.Printf("ERROR: create conversation: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusCreated, conv)
}

func (h *ConversationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	conv, err := h.service.GetByID(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.NotFound(w, "Conversation")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: get conversation %s: %v", id, err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, conv)
}

func (h *ConversationHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	var req updateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errs := req.validate(); len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	conv, err := h.service.Update(r.Context(), service.UpdateConversationInput{
		ConversationID: id,
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		AvatarURL:      req.AvatarURL,
	})
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.NotFound(w, "Conversation")
			return
		}
		if errors.Is(err, service.ErrCannotModifyDirect) {
			response.BadRequest(w, "Cannot modify a direct conversation")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.Forbidden(w, "Only owners and admins can modify this conversation")
			return
		}
		log.Printf("ERROR: update conversation %s: %v", id, err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, conv)
}

func (h *ConversationHandler) Archive(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	conv, err := h.service.Archive(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.NotFound(w, "Conversation")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.Forbidden(w, "Only the owner can archive this conversation")
			return
		}
		log.Printf("ERROR: archive conversation %s: %v", id, err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, conv)
}

func (h *ConversationHandler) AddParticipants(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	convID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	var req addParticipantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	participantIDs, errs := req.validate()
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	added, err := h.service.AddParticipants(r.Context(), service.AddParticipantsInput{
		ConversationID: convID,
		UserID:         userID,
		ParticipantIDs: participantIDs,
	})
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.NotFound(w, "Conversation")
			return
		}
		if errors.Is(err, service.ErrCannotModifyDirect) {
			response.BadRequest(w, "Cannot add participants to a direct conversation")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.Forbidden(w, "Only owners and admins can add participants")
			return
		}
		if errors.Is(err, service.ErrInvalidUserIDs) {
			response.ValidationError(w, map[string]string{"participant_ids": "one or more user IDs do not exist"})
			return
		}
		log.Printf("ERROR: add participants to conversation %s: %v", convID, err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusCreated, added)
}

func (h *ConversationHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	convID, err := uuid.Parse(vars["id"])
	if err != nil {
		response.BadRequest(w, "Invalid conversation ID format, expected UUID")
		return
	}

	targetUserID, err := uuid.Parse(vars["userId"])
	if err != nil {
		response.BadRequest(w, "Invalid user ID format, expected UUID")
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		response.BadRequest(w, "Missing or invalid X-User-ID header")
		return
	}

	err = h.service.RemoveParticipant(r.Context(), service.RemoveParticipantInput{
		ConversationID: convID,
		UserID:         userID,
		TargetUserID:   targetUserID,
	})
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.NotFound(w, "Conversation")
			return
		}
		if errors.Is(err, service.ErrCannotModifyDirect) {
			response.BadRequest(w, "Cannot remove participants from a direct conversation")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "User is not a participant of this conversation")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.Forbidden(w, "Insufficient permissions to remove this participant")
			return
		}
		log.Printf("ERROR: remove participant %s from conversation %s: %v", targetUserID, convID, err)
		response.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
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

	var search *string
	if s := r.URL.Query().Get("search"); s != "" {
		if len(s) > 100 {
			response.ValidationError(w, map[string]string{"search": "search must not exceed 100 characters"})
			return
		}
		search = &s
	}

	result, err := h.service.ListByUser(r.Context(), service.ListConversationsInput{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
		Search: search,
	})
	if err != nil {
		log.Printf("ERROR: list conversations for user %s: %v", userID, err)
		response.InternalError(w)
		return
	}

	response.SuccessWithPagination(w, result.Conversations, response.PaginationMeta{
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	})
}

// --- Helpers ---

func getUserID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.Header.Get("X-User-ID"))
}

func parsePagination(r *http.Request) (limit int32, offset int32, errs map[string]string) {
	errs = make(map[string]string)
	limit = 20
	offset = 0

	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			errs["limit"] = "limit must be a positive integer"
		} else if n > 100 {
			errs["limit"] = "limit must not exceed 100"
		} else {
			limit = int32(n)
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			errs["offset"] = "offset must be a non-negative integer"
		} else {
			offset = int32(n)
		}
	}

	return limit, offset, errs
}
