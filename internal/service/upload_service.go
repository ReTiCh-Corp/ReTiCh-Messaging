package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/retich-corp/messaging/internal/db"
	"github.com/retich-corp/messaging/internal/storage"
)

var (
	ErrFileTooLarge      = errors.New("file exceeds maximum size")
	ErrInvalidFileType   = errors.New("file type not allowed")
	ErrAttachmentNotFound = errors.New("attachment not found")
)

const (
	MaxImageSize = 10 * 1024 * 1024  // 10MB
	MaxFileSize  = 50 * 1024 * 1024  // 50MB
)

// --- DTOs ---

type AttachmentResponse struct {
	ID           uuid.UUID  `json:"id"`
	MessageID    uuid.UUID  `json:"message_id"`
	FileName     string     `json:"file_name"`
	FileType     string     `json:"file_type"`
	FileSize     int64      `json:"file_size"`
	FileURL      string     `json:"file_url"`
	ThumbnailURL *string    `json:"thumbnail_url,omitempty"`
	Width        *int32     `json:"width,omitempty"`
	Height       *int32     `json:"height,omitempty"`
	Duration     *int32     `json:"duration,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type CreateAttachmentInput struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	FileName  string
	FileType  string
	FileSize  int64
	FileURL   string
	Width     *int32
	Height    *int32
	Duration  *int32
}

type ListMediaInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Limit          int32
	Offset         int32
}

type ListMediaResult struct {
	Attachments []AttachmentResponse
	Total       int64
}

// --- Interface ---

type UploadService interface {
	CreateAttachment(ctx context.Context, input CreateAttachmentInput) (AttachmentResponse, error)
	ListAttachmentsByMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) ([]AttachmentResponse, error)
	ListMediaByConversation(ctx context.Context, input ListMediaInput) (ListMediaResult, error)
}

// --- Implementation ---

type uploadService struct {
	store   db.Store
	storage storage.FileStorage
}

func NewUploadService(store db.Store, fs storage.FileStorage) UploadService {
	return &uploadService{store: store, storage: fs}
}

func (s *uploadService) CreateAttachment(ctx context.Context, input CreateAttachmentInput) (AttachmentResponse, error) {
	// Get message to verify it exists and get created_at for composite FK
	msg, err := s.store.GetMessageByID(ctx, input.MessageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AttachmentResponse{}, ErrMessageNotFound
		}
		return AttachmentResponse{}, err
	}

	// Verify user is participant
	_, err = s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: msg.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AttachmentResponse{}, ErrNotParticipant
		}
		return AttachmentResponse{}, err
	}

	params := db.CreateAttachmentParams{
		MessageID:        input.MessageID,
		MessageCreatedAt: msg.CreatedAt,
		FileName:         input.FileName,
		FileType:         input.FileType,
		FileSize:         input.FileSize,
		FileUrl:          input.FileURL,
	}
	if input.Width != nil {
		params.Width = sql.NullInt32{Int32: *input.Width, Valid: true}
	}
	if input.Height != nil {
		params.Height = sql.NullInt32{Int32: *input.Height, Valid: true}
	}
	if input.Duration != nil {
		params.Duration = sql.NullInt32{Int32: *input.Duration, Valid: true}
	}

	attachment, err := s.store.CreateAttachment(ctx, params)
	if err != nil {
		return AttachmentResponse{}, err
	}

	return toAttachmentResponse(attachment), nil
}

func (s *uploadService) ListAttachmentsByMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) ([]AttachmentResponse, error) {
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

	attachments, err := s.store.ListAttachmentsByMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	results := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		results[i] = toAttachmentResponse(a)
	}
	return results, nil
}

func (s *uploadService) ListMediaByConversation(ctx context.Context, input ListMediaInput) (ListMediaResult, error) {
	_, err := s.store.GetParticipant(ctx, db.GetParticipantParams{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ListMediaResult{}, ErrNotParticipant
		}
		return ListMediaResult{}, err
	}

	attachments, err := s.store.ListAttachmentsByConversation(ctx, db.ListAttachmentsByConversationParams{
		ConversationID: input.ConversationID,
		Limit:          input.Limit,
		Offset:         input.Offset,
	})
	if err != nil {
		return ListMediaResult{}, err
	}

	total, err := s.store.CountAttachmentsByConversation(ctx, input.ConversationID)
	if err != nil {
		return ListMediaResult{}, err
	}

	results := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		results[i] = toAttachmentResponse(a)
	}

	return ListMediaResult{Attachments: results, Total: total}, nil
}

// --- Helpers ---

func toAttachmentResponse(a db.Attachment) AttachmentResponse {
	resp := AttachmentResponse{
		ID:        a.ID,
		MessageID: a.MessageID,
		FileName:  a.FileName,
		FileType:  a.FileType,
		FileSize:  a.FileSize,
		FileURL:   a.FileUrl,
	}
	if a.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &a.ThumbnailUrl.String
	}
	if a.Width.Valid {
		resp.Width = &a.Width.Int32
	}
	if a.Height.Valid {
		resp.Height = &a.Height.Int32
	}
	if a.Duration.Valid {
		resp.Duration = &a.Duration.Int32
	}
	if a.CreatedAt.Valid {
		resp.CreatedAt = a.CreatedAt.Time
	}
	return resp
}

// AllowedFileTypes returns the set of allowed MIME types.
var AllowedFileTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
	"video/mp4": true, "video/webm": true,
	"audio/mpeg": true, "audio/ogg": true, "audio/wav": true,
	"application/pdf": true,
	"text/plain": true,
	"application/zip": true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}
