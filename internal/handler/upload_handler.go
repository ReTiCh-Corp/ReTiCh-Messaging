package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/retich-corp/messaging/internal/response"
	"github.com/retich-corp/messaging/internal/service"
	"github.com/retich-corp/messaging/internal/storage"
)

type UploadHandler struct {
	service service.UploadService
	storage storage.FileStorage
}

func NewUploadHandler(svc service.UploadService, fs storage.FileStorage) *UploadHandler {
	return &UploadHandler{service: svc, storage: fs}
}

func (h *UploadHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/messages/{id}/attachments", h.Upload).Methods("POST")
	r.HandleFunc("/messages/{id}/attachments", h.ListByMessage).Methods("GET")
	r.HandleFunc("/conversations/{id}/media", h.ListMedia).Methods("GET")
}

// Upload handles multipart file upload and attaches it to a message.
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
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

	// Parse multipart form — max 50MB
	if err := r.ParseMultipartForm(service.MaxFileSize); err != nil {
		response.BadRequest(w, "File too large or invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Missing 'file' field in multipart form")
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback: infer from extension
		contentType = inferContentType(header.Filename)
	}
	if !service.AllowedFileTypes[contentType] {
		response.ValidationError(w, map[string]string{
			"file": "File type '" + contentType + "' is not allowed",
		})
		return
	}

	// Validate file size based on type
	maxSize := int64(service.MaxFileSize)
	if strings.HasPrefix(contentType, "image/") {
		maxSize = int64(service.MaxImageSize)
	}
	if header.Size > maxSize {
		response.ValidationError(w, map[string]string{
			"file": "File exceeds maximum allowed size",
		})
		return
	}

	// Save file to storage
	fileURL, err := h.storage.Save(header.Filename, file)
	if err != nil {
		log.Printf("ERROR: save file: %v", err)
		response.InternalError(w)
		return
	}

	// Create attachment record
	attachment, err := h.service.CreateAttachment(r.Context(), service.CreateAttachmentInput{
		MessageID: messageID,
		UserID:    userID,
		FileName:  header.Filename,
		FileType:  contentType,
		FileSize:  header.Size,
		FileURL:   fileURL,
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
		log.Printf("ERROR: create attachment: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusCreated, attachment)
}

// ListByMessage returns all attachments for a message.
func (h *UploadHandler) ListByMessage(w http.ResponseWriter, r *http.Request) {
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

	attachments, err := h.service.ListAttachmentsByMessage(r.Context(), messageID, userID)
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			response.NotFound(w, "Message")
			return
		}
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: list attachments: %v", err)
		response.InternalError(w)
		return
	}

	response.Success(w, http.StatusOK, attachments)
}

// ListMedia returns all media attachments for a conversation (gallery).
func (h *UploadHandler) ListMedia(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.service.ListMediaByConversation(r.Context(), service.ListMediaInput{
		ConversationID: convID,
		UserID:         userID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotParticipant) {
			response.Forbidden(w, "You are not a participant of this conversation")
			return
		}
		log.Printf("ERROR: list media: %v", err)
		response.InternalError(w)
		return
	}

	response.SuccessWithPagination(w, result.Attachments, response.PaginationMeta{
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	})
}

// inferContentType guesses MIME type from file extension.
func inferContentType(filename string) string {
	ext := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(ext, ".jpg"), strings.HasSuffix(ext, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(ext, ".png"):
		return "image/png"
	case strings.HasSuffix(ext, ".gif"):
		return "image/gif"
	case strings.HasSuffix(ext, ".webp"):
		return "image/webp"
	case strings.HasSuffix(ext, ".mp4"):
		return "video/mp4"
	case strings.HasSuffix(ext, ".webm"):
		return "video/webm"
	case strings.HasSuffix(ext, ".mp3"):
		return "audio/mpeg"
	case strings.HasSuffix(ext, ".pdf"):
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
