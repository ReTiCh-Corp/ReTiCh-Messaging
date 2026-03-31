package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/retich-corp/messaging/internal/db"
)

// --- Errors ---

var (
	ErrReactionAlreadyExists = errors.New("reaction already exists")
	ErrPinAlreadyExists      = errors.New("message is already pinned")
)

// --- DTOs ---

type ReactionResponse struct {
	ID        uuid.UUID `json:"id"`
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type ReadReceiptResponse struct {
	ID                uuid.UUID `json:"id"`
	ConversationID    uuid.UUID `json:"conversation_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
	LastReadAt        time.Time `json:"last_read_at"`
}

type PinnedMessageResponse struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
	PinnedBy       uuid.UUID `json:"pinned_by"`
	PinnedAt       time.Time `json:"pinned_at"`
}

// --- Inputs ---

type AddReactionInput struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
}

type RemoveReactionInput struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
}

type UpdateReadReceiptInput struct {
	ConversationID    uuid.UUID
	UserID            uuid.UUID
	LastReadMessageID uuid.UUID
}

type PinMessageInput struct {
	ConversationID uuid.UUID
	MessageID      uuid.UUID
	UserID         uuid.UUID
}

type UnpinMessageInput struct {
	ConversationID uuid.UUID
	MessageID      uuid.UUID
	UserID         uuid.UUID
}

// --- Interface ---

type SocialService interface {
	AddReaction(ctx context.Context, input AddReactionInput) (ReactionResponse, error)
	RemoveReaction(ctx context.Context, input RemoveReactionInput) error
	ListReactions(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) ([]ReactionResponse, error)

	UpdateReadReceipt(ctx context.Context, input UpdateReadReceiptInput) (ReadReceiptResponse, error)
	ListReadReceipts(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) ([]ReadReceiptResponse, error)

	PinMessage(ctx context.Context, input PinMessageInput) (PinnedMessageResponse, error)
	UnpinMessage(ctx context.Context, input UnpinMessageInput) error
	ListPinnedMessages(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) ([]PinnedMessageResponse, error)
}

// --- Implementation ---

type socialService struct {
	store       db.Store
	broadcaster EventBroadcaster
}

func NewSocialService(store db.Store, broadcaster EventBroadcaster) SocialService {
	return &socialService{store: store, broadcaster: broadcaster}
}

// --- Reactions ---

func (s *socialService) AddReaction(ctx context.Context, input AddReactionInput) (ReactionResponse, error) {
	// Get message to verify it exists and get conversation_id + created_at
	msg, err := s.store.GetMessageByID(ctx, input.MessageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReactionResponse{}, ErrMessageNotFound
		}
		return ReactionResponse{}, err
	}
	if msg.IsDeleted.Valid && msg.IsDeleted.Bool {
		return ReactionResponse{}, ErrMessageNotFound
	}

	// Verify user is participant
	_, err = s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: msg.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReactionResponse{}, ErrNotParticipant
		}
		return ReactionResponse{}, err
	}

	reaction, err := s.store.AddReaction(ctx, db.AddReactionParams{
		MessageID:        input.MessageID,
		MessageCreatedAt: msg.CreatedAt,
		UserID:           input.UserID,
		Emoji:            input.Emoji,
	})
	if err != nil {
		return ReactionResponse{}, err
	}

	// ON CONFLICT DO NOTHING returns zero rows — check if ID is zero
	if reaction.ID == uuid.Nil {
		return ReactionResponse{}, ErrReactionAlreadyExists
	}

	resp := toReactionResponse(reaction)

	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessageEvent("reaction.added", msg.ConversationID, resp)
	}

	return resp, nil
}

func (s *socialService) RemoveReaction(ctx context.Context, input RemoveReactionInput) error {
	msg, err := s.store.GetMessageByID(ctx, input.MessageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMessageNotFound
		}
		return err
	}

	err = s.store.RemoveReaction(ctx, db.RemoveReactionParams{
		MessageID: input.MessageID,
		UserID:    input.UserID,
		Emoji:     input.Emoji,
	})
	if err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessageEvent("reaction.removed", msg.ConversationID, map[string]interface{}{
			"message_id": input.MessageID,
			"user_id":    input.UserID,
			"emoji":      input.Emoji,
		})
	}

	return nil
}

func (s *socialService) ListReactions(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) ([]ReactionResponse, error) {
	msg, err := s.store.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	_, err = s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: msg.ConversationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	reactions, err := s.store.ListReactionsByMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	results := make([]ReactionResponse, len(reactions))
	for i, r := range reactions {
		results[i] = toReactionResponse(r)
	}
	return results, nil
}

// --- Read Receipts ---

func (s *socialService) UpdateReadReceipt(ctx context.Context, input UpdateReadReceiptInput) (ReadReceiptResponse, error) {
	// Verify user is participant
	_, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReadReceiptResponse{}, ErrNotParticipant
		}
		return ReadReceiptResponse{}, err
	}

	receipt, err := s.store.UpsertReadReceipt(ctx, db.UpsertReadReceiptParams{
		ConversationID:    input.ConversationID,
		UserID:            input.UserID,
		LastReadMessageID: input.LastReadMessageID,
	})
	if err != nil {
		return ReadReceiptResponse{}, err
	}

	resp := toReadReceiptResponse(receipt)

	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessageEvent("read.updated", input.ConversationID, resp)
	}

	return resp, nil
}

func (s *socialService) ListReadReceipts(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) ([]ReadReceiptResponse, error) {
	_, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	receipts, err := s.store.ListReadReceipts(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	results := make([]ReadReceiptResponse, len(receipts))
	for i, r := range receipts {
		results[i] = toReadReceiptResponse(r)
	}
	return results, nil
}

// --- Pinned Messages ---

func (s *socialService) PinMessage(ctx context.Context, input PinMessageInput) (PinnedMessageResponse, error) {
	msg, err := s.store.GetMessageByID(ctx, input.MessageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PinnedMessageResponse{}, ErrMessageNotFound
		}
		return PinnedMessageResponse{}, err
	}
	if msg.IsDeleted.Valid && msg.IsDeleted.Bool {
		return PinnedMessageResponse{}, ErrMessageNotFound
	}

	// Verify user is participant and has permission (admin/owner)
	participant, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PinnedMessageResponse{}, ErrNotParticipant
		}
		return PinnedMessageResponse{}, err
	}

	role := participant.Role.String
	if role != "owner" && role != "admin" {
		return PinnedMessageResponse{}, ErrForbidden
	}

	pin, err := s.store.PinMessage(ctx, db.PinMessageParams{
		ConversationID:   input.ConversationID,
		MessageID:        input.MessageID,
		MessageCreatedAt: msg.CreatedAt,
		PinnedBy:         input.UserID,
	})
	if err != nil {
		return PinnedMessageResponse{}, err
	}

	if pin.ID == uuid.Nil {
		return PinnedMessageResponse{}, ErrPinAlreadyExists
	}

	resp := toPinnedMessageResponse(pin)

	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessageEvent("message.pinned", input.ConversationID, resp)
	}

	return resp, nil
}

func (s *socialService) UnpinMessage(ctx context.Context, input UnpinMessageInput) error {
	// Verify user is participant and has permission
	participant, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotParticipant
		}
		return err
	}

	role := participant.Role.String
	if role != "owner" && role != "admin" {
		return ErrForbidden
	}

	err = s.store.UnpinMessage(ctx, db.UnpinMessageParams{
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
	})
	if err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessageEvent("message.unpinned", input.ConversationID, map[string]interface{}{
			"message_id":      input.MessageID,
			"conversation_id": input.ConversationID,
		})
	}

	return nil
}

func (s *socialService) ListPinnedMessages(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) ([]PinnedMessageResponse, error) {
	_, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	pins, err := s.store.ListPinnedMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	results := make([]PinnedMessageResponse, len(pins))
	for i, p := range pins {
		results[i] = toPinnedMessageResponse(p)
	}
	return results, nil
}

// --- Helpers ---

func toReactionResponse(r db.MessageReaction) ReactionResponse {
	resp := ReactionResponse{
		ID:        r.ID,
		MessageID: r.MessageID,
		UserID:    r.UserID,
		Emoji:     r.Emoji,
	}
	if r.CreatedAt.Valid {
		resp.CreatedAt = r.CreatedAt.Time
	}
	return resp
}

func toReadReceiptResponse(r db.ReadReceipt) ReadReceiptResponse {
	resp := ReadReceiptResponse{
		ID:                r.ID,
		ConversationID:    r.ConversationID,
		UserID:            r.UserID,
		LastReadMessageID: r.LastReadMessageID,
	}
	if r.LastReadAt.Valid {
		resp.LastReadAt = r.LastReadAt.Time
	}
	return resp
}

func toPinnedMessageResponse(p db.PinnedMessage) PinnedMessageResponse {
	resp := PinnedMessageResponse{
		ID:             p.ID,
		ConversationID: p.ConversationID,
		MessageID:      p.MessageID,
		PinnedBy:       p.PinnedBy,
	}
	if p.PinnedAt.Valid {
		resp.PinnedAt = p.PinnedAt.Time
	}
	return resp
}
