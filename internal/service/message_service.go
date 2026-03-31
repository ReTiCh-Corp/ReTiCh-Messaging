package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	db "github.com/retich-corp/messaging/internal/db"
)

var (
	ErrMessageNotFound  = errors.New("message not found")
	ErrNotMessageAuthor = errors.New("user is not the author of this message")
)

// --- DTOs ---

type MessageResponse struct {
	ID             uuid.UUID        `json:"id"`
	ConversationID uuid.UUID        `json:"conversation_id"`
	SenderID       uuid.UUID        `json:"sender_id"`
	Type           string           `json:"type"`
	Content        *string          `json:"content"`
	Metadata       *json.RawMessage `json:"metadata,omitempty"`
	ReplyToID      *uuid.UUID       `json:"reply_to_id,omitempty"`
	IsEdited       bool             `json:"is_edited"`
	EditedAt       *time.Time       `json:"edited_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

type CreateMessageInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Type           string
	Content        *string
	Metadata       *json.RawMessage
	ReplyToID      *uuid.UUID
}

type UpdateMessageInput struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Content   string
}

type ListMessagesInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Limit          int32
	Offset         int32
}

type ListMessagesResult struct {
	Messages []MessageResponse
	Total    int64
}

// --- Interface ---

type MessageService interface {
	Create(ctx context.Context, input CreateMessageInput) (MessageResponse, error)
	GetByID(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) (MessageResponse, error)
	ListByConversation(ctx context.Context, input ListMessagesInput) (ListMessagesResult, error)
	Update(ctx context.Context, input UpdateMessageInput) (MessageResponse, error)
	Delete(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error
}

// --- Implementation ---

type messageService struct {
	store db.Store
}

func NewMessageService(store db.Store) MessageService {
	return &messageService{store: store}
}

func (s *messageService) Create(ctx context.Context, input CreateMessageInput) (MessageResponse, error) {
	// Verify conversation exists
	_, err := s.store.GetConversationByID(ctx, input.ConversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MessageResponse{}, ErrConversationNotFound
		}
		return MessageResponse{}, err
	}

	// Verify sender is a participant
	_, err = s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.SenderID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MessageResponse{}, ErrNotParticipant
		}
		return MessageResponse{}, err
	}

	// Build params
	msgType := "text"
	if input.Type != "" {
		msgType = input.Type
	}

	params := db.CreateMessageParams{
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		Type:           db.NullMessageType{MessageType: db.MessageType(msgType), Valid: true},
	}
	if input.Content != nil {
		params.Content = sql.NullString{String: *input.Content, Valid: true}
	}
	if input.Metadata != nil {
		params.Metadata.RawMessage = *input.Metadata
		params.Metadata.Valid = true
	}
	if input.ReplyToID != nil {
		params.ReplyToID = uuid.NullUUID{UUID: *input.ReplyToID, Valid: true}
	}

	msg, err := s.store.CreateMessage(ctx, params)
	if err != nil {
		return MessageResponse{}, err
	}

	return toMessageResponse(msg), nil
}

func (s *messageService) GetByID(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) (MessageResponse, error) {
	msg, err := s.store.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MessageResponse{}, ErrMessageNotFound
		}
		return MessageResponse{}, err
	}

	if msg.IsDeleted.Valid && msg.IsDeleted.Bool {
		return MessageResponse{}, ErrMessageNotFound
	}

	// Verify user is a participant of this conversation
	_, err = s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: msg.ConversationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MessageResponse{}, ErrNotParticipant
		}
		return MessageResponse{}, err
	}

	return toMessageResponse(msg), nil
}

func (s *messageService) ListByConversation(ctx context.Context, input ListMessagesInput) (ListMessagesResult, error) {
	// Verify conversation exists
	_, err := s.store.GetConversationByID(ctx, input.ConversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ListMessagesResult{}, ErrConversationNotFound
		}
		return ListMessagesResult{}, err
	}

	// Verify user is a participant
	_, err = s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ListMessagesResult{}, ErrNotParticipant
		}
		return ListMessagesResult{}, err
	}

	messages, err := s.store.ListMessagesByConversation(ctx, db.ListMessagesByConversationParams{
		ConversationID: input.ConversationID,
		Limit:          input.Limit,
		Offset:         input.Offset,
	})
	if err != nil {
		return ListMessagesResult{}, err
	}

	total, err := s.store.CountMessagesByConversation(ctx, input.ConversationID)
	if err != nil {
		return ListMessagesResult{}, err
	}

	results := make([]MessageResponse, len(messages))
	for i, m := range messages {
		results[i] = toMessageResponse(m)
	}

	return ListMessagesResult{
		Messages: results,
		Total:    total,
	}, nil
}

func (s *messageService) Update(ctx context.Context, input UpdateMessageInput) (MessageResponse, error) {
	// Try the update (sender check is in SQL)
	updated, err := s.store.UpdateMessageContent(ctx, db.UpdateMessageContentParams{
		ID:       input.MessageID,
		SenderID: input.UserID,
		Content:  sql.NullString{String: input.Content, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Distinguish "not found" from "not author"
			msg, getErr := s.store.GetMessageByID(ctx, input.MessageID)
			if getErr != nil {
				if errors.Is(getErr, sql.ErrNoRows) {
					return MessageResponse{}, ErrMessageNotFound
				}
				return MessageResponse{}, getErr
			}
			if msg.IsDeleted.Valid && msg.IsDeleted.Bool {
				return MessageResponse{}, ErrMessageNotFound
			}
			if msg.SenderID != input.UserID {
				return MessageResponse{}, ErrNotMessageAuthor
			}
			// Should not reach here, but log and return generic error
			log.Printf("ERROR: update message %s: unexpected ErrNoRows with matching sender", input.MessageID)
			return MessageResponse{}, err
		}
		return MessageResponse{}, err
	}

	return toMessageResponse(updated), nil
}

func (s *messageService) Delete(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error {
	// Get the message
	msg, err := s.store.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMessageNotFound
		}
		return err
	}

	if msg.IsDeleted.Valid && msg.IsDeleted.Bool {
		return ErrMessageNotFound
	}

	// Verify user is a participant
	participant, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: msg.ConversationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotParticipant
		}
		return err
	}

	// Authorization: author, admin, or owner can delete
	if msg.SenderID != userID {
		role := participant.Role.String
		if role != "owner" && role != "admin" {
			return ErrForbidden
		}
	}

	_, err = s.store.SoftDeleteMessage(ctx, messageID)
	return err
}

// --- Helpers ---

func toMessageResponse(m db.Message) MessageResponse {
	resp := MessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		IsEdited:       m.IsEdited.Valid && m.IsEdited.Bool,
		CreatedAt:      m.CreatedAt,
	}
	if m.Type.Valid {
		resp.Type = string(m.Type.MessageType)
	}
	if m.Content.Valid {
		resp.Content = &m.Content.String
	}
	if m.Metadata.Valid {
		raw := json.RawMessage(m.Metadata.RawMessage)
		resp.Metadata = &raw
	}
	if m.ReplyToID.Valid {
		resp.ReplyToID = &m.ReplyToID.UUID
	}
	if m.EditedAt.Valid {
		resp.EditedAt = &m.EditedAt.Time
	}
	return resp
}
