package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	db "github.com/retich-corp/messaging/internal/db"
)

var (
	ErrConversationNotFound         = errors.New("conversation not found")
	ErrNotParticipant               = errors.New("user is not a participant of this conversation")
	ErrForbidden                    = errors.New("insufficient permissions")
	ErrDirectConversationExists     = errors.New("direct conversation already exists between these users")
	ErrCannotModifyDirect           = errors.New("cannot modify a direct conversation")
	ErrInvalidUserIDs               = errors.New("one or more user IDs are invalid")
	ErrDirectRequiresOneParticipant = errors.New("direct conversation requires exactly one other participant")
)

// UserValidator validates that user IDs exist via an external service.
type UserValidator interface {
	ValidateUserIDs(ctx context.Context, userIDs []uuid.UUID) (invalidIDs []uuid.UUID, err error)
}

type ConversationResponse struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Name          *string    `json:"name"`
	Description   *string    `json:"description"`
	AvatarURL     *string    `json:"avatar_url"`
	CreatorID     *uuid.UUID `json:"creator_id"`
	IsArchived    bool       `json:"is_archived"`
	LastMessageAt *time.Time `json:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	UnreadCount   int64      `json:"unread_count"`
}

type ParticipantResponse struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	Role           string    `json:"role"`
	Nickname       *string   `json:"nickname"`
	JoinedAt       time.Time `json:"joined_at"`
}

type ConversationDetailResponse struct {
	ConversationResponse
	Participants []ParticipantResponse `json:"participants"`
}

type CreateConversationInput struct {
	Type           string
	Name           *string
	Description    *string
	AvatarURL      *string
	CreatorID      uuid.UUID
	ParticipantIDs []uuid.UUID
}

type UpdateConversationInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Name           *string
	Description    *string
	AvatarURL      *string
}

type AddParticipantsInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	ParticipantIDs []uuid.UUID
}

type RemoveParticipantInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	TargetUserID   uuid.UUID
}

type ListConversationsInput struct {
	UserID uuid.UUID
	Limit  int32
	Offset int32
	Search *string
}

type ListConversationsResult struct {
	Conversations []ConversationResponse
	Total         int64
}

// ExistingConversationError wraps ErrDirectConversationExists with the existing conversation data.
type ExistingConversationError struct {
	Conversation ConversationDetailResponse
}

func (e *ExistingConversationError) Error() string {
	return ErrDirectConversationExists.Error()
}

func (e *ExistingConversationError) Unwrap() error {
	return ErrDirectConversationExists
}

type ConversationService interface {
	Create(ctx context.Context, input CreateConversationInput) (ConversationDetailResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (ConversationDetailResponse, error)
	ListByUser(ctx context.Context, input ListConversationsInput) (ListConversationsResult, error)
	Update(ctx context.Context, input UpdateConversationInput) (ConversationDetailResponse, error)
	Archive(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (ConversationResponse, error)
	AddParticipants(ctx context.Context, input AddParticipantsInput) ([]ParticipantResponse, error)
	RemoveParticipant(ctx context.Context, input RemoveParticipantInput) error
}

type conversationService struct {
	store         db.Store
	userValidator UserValidator
}

func NewConversationService(store db.Store, userValidator UserValidator) ConversationService {
	return &conversationService{
		store:         store,
		userValidator: userValidator,
	}
}

func (s *conversationService) Create(ctx context.Context, input CreateConversationInput) (ConversationDetailResponse, error) {
	// Validate user IDs exist via User service
	if s.userValidator != nil {
		invalidIDs, err := s.userValidator.ValidateUserIDs(ctx, input.ParticipantIDs)
		if err != nil {
			return ConversationDetailResponse{}, fmt.Errorf("user validation failed: %w", err)
		}
		if len(invalidIDs) > 0 {
			return ConversationDetailResponse{}, ErrInvalidUserIDs
		}
	}

	// For direct conversations, check if one already exists
	if input.Type == "direct" {
		if len(input.ParticipantIDs) != 1 {
			return ConversationDetailResponse{}, ErrDirectRequiresOneParticipant
		}

		existing, err := s.store.GetDirectConversationBetweenUsers(ctx, db.GetDirectConversationBetweenUsersParams{
			UserID:   input.CreatorID,
			UserID_2: input.ParticipantIDs[0],
		})
		if err == nil {
			// Conversation already exists, return it with participants
			participants, _ := s.store.ListParticipants(ctx, existing.ID)
			return ConversationDetailResponse{}, &ExistingConversationError{
				Conversation: ConversationDetailResponse{
					ConversationResponse: toConversationResponse(existing),
					Participants:         toParticipantResponses(participants),
				},
			}
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return ConversationDetailResponse{}, err
		}
	}

	// Execute creation within a transaction
	var conv db.Conversation
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		// Create conversation
		params := db.CreateConversationParams{
			Type: db.ConversationType(input.Type),
		}
		if input.Type != "direct" {
			params.CreatorID = uuid.NullUUID{UUID: input.CreatorID, Valid: true}
		}
		if input.Name != nil {
			params.Name = sql.NullString{String: *input.Name, Valid: true}
		}
		if input.Description != nil {
			params.Description = sql.NullString{String: *input.Description, Valid: true}
		}
		if input.AvatarURL != nil {
			params.AvatarUrl = sql.NullString{String: *input.AvatarURL, Valid: true}
		}

		var txErr error
		conv, txErr = q.CreateConversation(ctx, params)
		if txErr != nil {
			return txErr
		}

		// Add creator as participant
		creatorRole := "owner"
		if input.Type == "direct" {
			creatorRole = "member"
		}
		_, txErr = q.AddParticipant(ctx, db.AddParticipantParams{
			ConversationID: conv.ID,
			UserID:         input.CreatorID,
			Role:           sql.NullString{String: creatorRole, Valid: true},
		})
		if txErr != nil {
			return txErr
		}

		// Add other participants
		for _, pid := range input.ParticipantIDs {
			_, txErr = q.AddParticipant(ctx, db.AddParticipantParams{
				ConversationID: conv.ID,
				UserID:         pid,
				Role:           sql.NullString{String: "member", Valid: true},
			})
			if txErr != nil {
				return txErr
			}
		}

		return nil
	})
	if err != nil {
		return ConversationDetailResponse{}, err
	}

	// Fetch participants for response
	participants, err := s.store.ListParticipants(ctx, conv.ID)
	if err != nil {
		return ConversationDetailResponse{}, err
	}

	return ConversationDetailResponse{
		ConversationResponse: toConversationResponse(conv),
		Participants:         toParticipantResponses(participants),
	}, nil
}

func (s *conversationService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (ConversationDetailResponse, error) {
	conv, err := s.store.GetConversationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationDetailResponse{}, ErrConversationNotFound
		}
		return ConversationDetailResponse{}, err
	}

	// Check user is a participant
	_, err = s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: id,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationDetailResponse{}, ErrNotParticipant
		}
		return ConversationDetailResponse{}, err
	}

	participants, err := s.store.ListParticipants(ctx, id)
	if err != nil {
		return ConversationDetailResponse{}, err
	}

	return ConversationDetailResponse{
		ConversationResponse: toConversationResponse(conv),
		Participants:         toParticipantResponses(participants),
	}, nil
}

func (s *conversationService) ListByUser(ctx context.Context, input ListConversationsInput) (ListConversationsResult, error) {
	if input.Search != nil && *input.Search != "" {
		return s.searchByUser(ctx, input)
	}

	conversations, err := s.store.ListConversationsByUser(ctx, db.ListConversationsByUserParams{
		UserID: input.UserID,
		Limit:  input.Limit,
		Offset: input.Offset,
	})
	if err != nil {
		return ListConversationsResult{}, err
	}

	total, err := s.store.CountConversationsByUser(ctx, input.UserID)
	if err != nil {
		return ListConversationsResult{}, err
	}

	results := make([]ConversationResponse, len(conversations))
	for i, c := range conversations {
		results[i] = toConversationResponseFromRow(c.ID, c.Type, c.Name, c.Description, c.AvatarUrl, c.CreatorID, c.IsArchived, c.LastMessageAt, c.CreatedAt, c.UpdatedAt, c.UnreadCount)
	}
	return ListConversationsResult{Conversations: results, Total: total}, nil
}

func (s *conversationService) Update(ctx context.Context, input UpdateConversationInput) (ConversationDetailResponse, error) {
	conv, err := s.store.GetConversationByID(ctx, input.ConversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationDetailResponse{}, ErrConversationNotFound
		}
		return ConversationDetailResponse{}, err
	}

	if conv.Type == db.ConversationTypeDirect {
		return ConversationDetailResponse{}, ErrCannotModifyDirect
	}

	// Check authorization
	participant, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationDetailResponse{}, ErrNotParticipant
		}
		return ConversationDetailResponse{}, err
	}

	role := participant.Role.String
	if role != "owner" && role != "admin" {
		return ConversationDetailResponse{}, ErrForbidden
	}

	// Build update params - keep existing values for fields not provided
	updateParams := db.UpdateConversationParams{
		ID:          input.ConversationID,
		Name:        conv.Name,
		Description: conv.Description,
		AvatarUrl:   conv.AvatarUrl,
	}
	if input.Name != nil {
		updateParams.Name = sql.NullString{String: *input.Name, Valid: true}
	}
	if input.Description != nil {
		updateParams.Description = sql.NullString{String: *input.Description, Valid: true}
	}
	if input.AvatarURL != nil {
		updateParams.AvatarUrl = sql.NullString{String: *input.AvatarURL, Valid: true}
	}

	updated, err := s.store.UpdateConversation(ctx, updateParams)
	if err != nil {
		return ConversationDetailResponse{}, err
	}

	participants, err := s.store.ListParticipants(ctx, input.ConversationID)
	if err != nil {
		return ConversationDetailResponse{}, err
	}

	return ConversationDetailResponse{
		ConversationResponse: toConversationResponse(updated),
		Participants:         toParticipantResponses(participants),
	}, nil
}

func (s *conversationService) Archive(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (ConversationResponse, error) {
	_, err := s.store.GetConversationByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationResponse{}, ErrConversationNotFound
		}
		return ConversationResponse{}, err
	}

	participant, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationResponse{}, ErrNotParticipant
		}
		return ConversationResponse{}, err
	}

	if participant.Role.String != "owner" {
		return ConversationResponse{}, ErrForbidden
	}

	archived, err := s.store.ArchiveConversation(ctx, conversationID)
	if err != nil {
		return ConversationResponse{}, err
	}

	return toConversationResponse(archived), nil
}

func (s *conversationService) AddParticipants(ctx context.Context, input AddParticipantsInput) ([]ParticipantResponse, error) {
	conv, err := s.store.GetConversationByID(ctx, input.ConversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	if conv.Type == db.ConversationTypeDirect {
		return nil, ErrCannotModifyDirect
	}

	participant, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	role := participant.Role.String
	if role != "owner" && role != "admin" {
		return nil, ErrForbidden
	}

	// Validate user IDs exist
	if s.userValidator != nil {
		invalidIDs, err := s.userValidator.ValidateUserIDs(ctx, input.ParticipantIDs)
		if err != nil {
			return nil, fmt.Errorf("user validation failed: %w", err)
		}
		if len(invalidIDs) > 0 {
			return nil, ErrInvalidUserIDs
		}
	}

	var added []ParticipantResponse
	for _, pid := range input.ParticipantIDs {
		p, err := s.store.AddParticipant(ctx, db.AddParticipantParams{
			ConversationID: input.ConversationID,
			UserID:         pid,
			Role:           sql.NullString{String: "member", Valid: true},
		})
		if err != nil {
			// Skip duplicates (unique constraint violation)
			continue
		}
		added = append(added, toParticipantResponse(p))
	}

	return added, nil
}

func (s *conversationService) RemoveParticipant(ctx context.Context, input RemoveParticipantInput) error {
	conv, err := s.store.GetConversationByID(ctx, input.ConversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConversationNotFound
		}
		return err
	}

	isSelfLeave := input.UserID == input.TargetUserID

	if isSelfLeave {
		// Verify the user is actually a participant
		_, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
			ConversationID: input.ConversationID,
			UserID:         input.UserID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotParticipant
			}
			return err
		}
	} else {
		// Removing another user
		if conv.Type == db.ConversationTypeDirect {
			return ErrCannotModifyDirect
		}

		// Check requester permissions
		requester, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
			ConversationID: input.ConversationID,
			UserID:         input.UserID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotParticipant
			}
			return err
		}

		requesterRole := requester.Role.String
		if requesterRole != "owner" && requesterRole != "admin" {
			return ErrForbidden
		}

		// Check target exists
		target, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
			ConversationID: input.ConversationID,
			UserID:         input.TargetUserID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotParticipant
			}
			return err
		}

		targetRole := target.Role.String
		if targetRole == "owner" {
			return ErrForbidden
		}
		if requesterRole == "admin" && targetRole == "admin" {
			return ErrForbidden
		}
	}

	return s.store.RemoveParticipant(ctx, db.RemoveParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.TargetUserID,
	})
}

func (s *conversationService) searchByUser(ctx context.Context, input ListConversationsInput) (ListConversationsResult, error) {
	searchTerm := sql.NullString{String: *input.Search, Valid: true}

	conversations, err := s.store.SearchConversationsByUser(ctx, db.SearchConversationsByUserParams{
		UserID:     input.UserID,
		SearchTerm: searchTerm,
		Limit:      input.Limit,
		Offset:     input.Offset,
	})
	if err != nil {
		return ListConversationsResult{}, err
	}

	total, err := s.store.CountSearchConversationsByUser(ctx, db.CountSearchConversationsByUserParams{
		UserID:     input.UserID,
		SearchTerm: searchTerm,
	})
	if err != nil {
		return ListConversationsResult{}, err
	}

	results := make([]ConversationResponse, len(conversations))
	for i, c := range conversations {
		results[i] = toConversationResponseFromRow(c.ID, c.Type, c.Name, c.Description, c.AvatarUrl, c.CreatorID, c.IsArchived, c.LastMessageAt, c.CreatedAt, c.UpdatedAt, c.UnreadCount)
	}
	return ListConversationsResult{Conversations: results, Total: total}, nil
}

func toListResult(conversations []db.Conversation, total int64) ListConversationsResult {
	results := make([]ConversationResponse, len(conversations))
	for i, c := range conversations {
		results[i] = toConversationResponse(c)
	}
	return ListConversationsResult{
		Conversations: results,
		Total:         total,
	}
}

func toConversationResponse(c db.Conversation) ConversationResponse {
	resp := ConversationResponse{
		ID:         c.ID,
		Type:       string(c.Type),
		IsArchived: c.IsArchived.Bool,
	}

	if c.Name.Valid {
		resp.Name = &c.Name.String
	}
	if c.Description.Valid {
		resp.Description = &c.Description.String
	}
	if c.AvatarUrl.Valid {
		resp.AvatarURL = &c.AvatarUrl.String
	}
	if c.CreatorID.Valid {
		resp.CreatorID = &c.CreatorID.UUID
	}
	if c.LastMessageAt.Valid {
		resp.LastMessageAt = &c.LastMessageAt.Time
	}
	if c.CreatedAt.Valid {
		resp.CreatedAt = c.CreatedAt.Time
	}
	if c.UpdatedAt.Valid {
		resp.UpdatedAt = c.UpdatedAt.Time
	}

	return resp
}

func toConversationResponseFromRow(
	id uuid.UUID, convType db.ConversationType, name, description, avatarUrl sql.NullString,
	creatorID uuid.NullUUID, isArchived sql.NullBool, lastMessageAt, createdAt, updatedAt sql.NullTime,
	unreadCount int64,
) ConversationResponse {
	resp := ConversationResponse{
		ID:          id,
		Type:        string(convType),
		IsArchived:  isArchived.Bool,
		UnreadCount: unreadCount,
	}
	if name.Valid {
		resp.Name = &name.String
	}
	if description.Valid {
		resp.Description = &description.String
	}
	if avatarUrl.Valid {
		resp.AvatarURL = &avatarUrl.String
	}
	if creatorID.Valid {
		resp.CreatorID = &creatorID.UUID
	}
	if lastMessageAt.Valid {
		resp.LastMessageAt = &lastMessageAt.Time
	}
	if createdAt.Valid {
		resp.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		resp.UpdatedAt = updatedAt.Time
	}
	return resp
}

func toParticipantResponse(p db.ConversationParticipant) ParticipantResponse {
	resp := ParticipantResponse{
		ID:             p.ID,
		ConversationID: p.ConversationID,
		UserID:         p.UserID,
		Role:           p.Role.String,
	}
	if p.Nickname.Valid {
		resp.Nickname = &p.Nickname.String
	}
	if p.JoinedAt.Valid {
		resp.JoinedAt = p.JoinedAt.Time
	}
	return resp
}

func toParticipantResponses(participants []db.ConversationParticipant) []ParticipantResponse {
	results := make([]ParticipantResponse, len(participants))
	for i, p := range participants {
		results[i] = toParticipantResponse(p)
	}
	return results
}
