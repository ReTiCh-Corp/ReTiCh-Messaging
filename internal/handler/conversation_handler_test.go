package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/retich-corp/messaging/internal/response"
	"github.com/retich-corp/messaging/internal/service"
)

// --- Helpers ---

func sampleConversationDetail() service.ConversationDetailResponse {
	name := "Test Group"
	creatorID := uuid.New()
	return service.ConversationDetailResponse{
		ConversationResponse: service.ConversationResponse{
			ID:        uuid.New(),
			Type:      "group",
			Name:      &name,
			CreatorID: &creatorID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Participants: []service.ParticipantResponse{
			{ID: uuid.New(), UserID: creatorID, Role: "owner", JoinedAt: time.Now()},
		},
	}
}

func newTestRouter(h *ConversationHandler) *mux.Router {
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func makeRequest(method, url, body string, userID string) *http.Request {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}
	return req
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()
	var resp response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("could not decode response: %v\nbody: %s", err, w.Body.String())
	}
	return resp
}

// =============================================================================
// Create
// =============================================================================

func TestCreate_201_Group(t *testing.T) {
	mock := &mockConversationService{
		createResult: sampleConversationDetail(),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	userID := uuid.New().String()
	body := `{"type":"group","name":"Test","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_201_Direct(t *testing.T) {
	mock := &mockConversationService{
		createResult: service.ConversationDetailResponse{
			ConversationResponse: service.ConversationResponse{
				ID:   uuid.New(),
				Type: "direct",
			},
		},
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	userID := uuid.New().String()
	body := `{"type":"direct","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_400_InvalidJSON(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("POST", "/conversations", "not json", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_MissingType(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_DirectWithName(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"direct","name":"bad","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_DirectWithDescription(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"direct","description":"bad","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_NoParticipants(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"group","name":"Test","participant_ids":[]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_CreatorInParticipants(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	userID := uuid.New().String()
	body := `{"type":"group","name":"Test","participant_ids":["` + userID + `"]}`
	req := makeRequest("POST", "/conversations", body, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_InvalidParticipantUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"group","name":"Test","participant_ids":["not-a-uuid"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_DirectMultipleParticipants(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"direct","participant_ids":["` + uuid.New().String() + `","` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_GroupMissingName(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"group","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_NameTooLong(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	longName := strings.Repeat("a", 101)
	body := `{"type":"group","name":"` + longName + `","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_AvatarURLTooLong(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	longURL := strings.Repeat("a", 501)
	body := `{"type":"group","name":"Test","avatar_url":"` + longURL + `","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_InvalidType(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"invalid","name":"Test","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_MissingUserID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"type":"group","name":"Test","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_409_DirectExists(t *testing.T) {
	existingConv := sampleConversationDetail()
	mock := &mockConversationService{
		createErr: &service.ExistingConversationError{Conversation: existingConv},
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"type":"direct","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_400_DirectRequiresOneParticipant(t *testing.T) {
	mock := &mockConversationService{
		createErr: service.ErrDirectRequiresOneParticipant,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"type":"direct","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_400_InvalidUserIDs(t *testing.T) {
	mock := &mockConversationService{
		createErr: service.ErrInvalidUserIDs,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"type":"group","name":"Test","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_500_InternalError(t *testing.T) {
	mock := &mockConversationService{
		createErr: service.ErrConversationNotFound, // arbitrary non-mapped error
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"type":"group","name":"Test","participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// GetByID
// =============================================================================

func TestGetByID_200(t *testing.T) {
	mock := &mockConversationService{
		getByIDResult: sampleConversationDetail(),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetByID_400_InvalidUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations/not-a-uuid", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetByID_400_MissingUserID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations/"+uuid.New().String(), "", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetByID_404(t *testing.T) {
	mock := &mockConversationService{
		getByIDErr: service.ErrConversationNotFound,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetByID_403(t *testing.T) {
	mock := &mockConversationService{
		getByIDErr: service.ErrNotParticipant,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestGetByID_500(t *testing.T) {
	mock := &mockConversationService{
		getByIDErr: errors.New("unexpected db error"),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// Update
// =============================================================================

func TestUpdate_200(t *testing.T) {
	mock := &mockConversationService{
		updateResult: sampleConversationDetail(),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"name":"Updated Name"}`
	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdate_400_InvalidJSON(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), "not json", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_400_NoFields(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_400_EmptyName(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":""}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_400_NameTooLong(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	longName := strings.Repeat("a", 101)
	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":"`+longName+`"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_400_AvatarURLTooLong(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	longURL := strings.Repeat("a", 501)
	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"avatar_url":"`+longURL+`"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_400_InvalidUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/not-uuid", `{"name":"ok"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_400_MissingUserID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":"ok"}`, "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_400_CannotModifyDirect(t *testing.T) {
	mock := &mockConversationService{
		updateErr: service.ErrCannotModifyDirect,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":"ok"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_403_NotParticipant(t *testing.T) {
	mock := &mockConversationService{
		updateErr: service.ErrNotParticipant,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":"ok"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestUpdate_403_Forbidden(t *testing.T) {
	mock := &mockConversationService{
		updateErr: service.ErrForbidden,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":"ok"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestUpdate_404(t *testing.T) {
	mock := &mockConversationService{
		updateErr: service.ErrConversationNotFound,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":"ok"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdate_500(t *testing.T) {
	mock := &mockConversationService{
		updateErr: errors.New("unexpected db error"),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("PUT", "/conversations/"+uuid.New().String(), `{"name":"ok"}`, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// Archive
// =============================================================================

func TestArchive_200(t *testing.T) {
	mock := &mockConversationService{
		archiveResult: service.ConversationResponse{ID: uuid.New(), Type: "group", IsArchived: true},
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestArchive_400_InvalidUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/not-uuid", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestArchive_400_MissingUserID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String(), "", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestArchive_403_NotParticipant(t *testing.T) {
	mock := &mockConversationService{
		archiveErr: service.ErrNotParticipant,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestArchive_403_NotOwner(t *testing.T) {
	mock := &mockConversationService{
		archiveErr: service.ErrForbidden,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestArchive_404(t *testing.T) {
	mock := &mockConversationService{
		archiveErr: service.ErrConversationNotFound,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArchive_500(t *testing.T) {
	mock := &mockConversationService{
		archiveErr: errors.New("unexpected db error"),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// AddParticipants
// =============================================================================

func TestAddParticipants_201(t *testing.T) {
	mock := &mockConversationService{
		addPartResult: []service.ParticipantResponse{
			{ID: uuid.New(), UserID: uuid.New(), Role: "member"},
		},
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddParticipants_400_InvalidJSON(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", "not json", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddParticipants_400_Empty(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"participant_ids":[]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddParticipants_400_TooMany(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	ids := make([]string, 51)
	for i := range ids {
		ids[i] = `"` + uuid.New().String() + `"`
	}
	body := `{"participant_ids":[` + strings.Join(ids, ",") + `]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddParticipants_400_InvalidUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"participant_ids":["not-a-uuid"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddParticipants_400_InvalidConvUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/not-uuid/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddParticipants_400_MissingUserID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddParticipants_403(t *testing.T) {
	mock := &mockConversationService{
		addPartErr: service.ErrForbidden,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAddParticipants_404(t *testing.T) {
	mock := &mockConversationService{
		addPartErr: service.ErrConversationNotFound,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAddParticipants_403_NotParticipant(t *testing.T) {
	mock := &mockConversationService{
		addPartErr: service.ErrNotParticipant,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAddParticipants_400_InvalidUserIDs(t *testing.T) {
	mock := &mockConversationService{
		addPartErr: service.ErrInvalidUserIDs,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddParticipants_500(t *testing.T) {
	mock := &mockConversationService{
		addPartErr: errors.New("unexpected db error"),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestAddParticipants_400_CannotModifyDirect(t *testing.T) {
	mock := &mockConversationService{
		addPartErr: service.ErrCannotModifyDirect,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	body := `{"participant_ids":["` + uuid.New().String() + `"]}`
	req := makeRequest("POST", "/conversations/"+uuid.New().String()+"/participants", body, uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// =============================================================================
// RemoveParticipant
// =============================================================================

func TestRemoveParticipant_204(t *testing.T) {
	mock := &mockConversationService{}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveParticipant_204_Leave(t *testing.T) {
	mock := &mockConversationService{}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	userID := uuid.New().String()
	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+userID, "", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveParticipant_400_InvalidConvUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/not-uuid/participants/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRemoveParticipant_400_InvalidUserUUID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/not-uuid", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRemoveParticipant_400_MissingUserID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+uuid.New().String(), "", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRemoveParticipant_403(t *testing.T) {
	mock := &mockConversationService{
		removePartErr: service.ErrForbidden,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRemoveParticipant_404(t *testing.T) {
	mock := &mockConversationService{
		removePartErr: service.ErrConversationNotFound,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRemoveParticipant_403_NotParticipant(t *testing.T) {
	mock := &mockConversationService{
		removePartErr: service.ErrNotParticipant,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRemoveParticipant_500(t *testing.T) {
	mock := &mockConversationService{
		removePartErr: errors.New("unexpected db error"),
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRemoveParticipant_400_CannotModifyDirect(t *testing.T) {
	mock := &mockConversationService{
		removePartErr: service.ErrCannotModifyDirect,
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("DELETE", "/conversations/"+uuid.New().String()+"/participants/"+uuid.New().String(), "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// =============================================================================
// List
// =============================================================================

func TestList_200(t *testing.T) {
	mock := &mockConversationService{
		listResult: service.ListConversationsResult{
			Conversations: []service.ConversationResponse{
				{ID: uuid.New(), Type: "group"},
			},
			Total: 1,
		},
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestList_200_WithSearch(t *testing.T) {
	mock := &mockConversationService{
		listResult: service.ListConversationsResult{
			Conversations: []service.ConversationResponse{},
			Total:         0,
		},
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations?search=test&limit=10&offset=5", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestList_400_InvalidLimit(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations?limit=abc", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestList_400_LimitTooHigh(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations?limit=101", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestList_400_NegativeOffset(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations?offset=-1", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestList_400_SearchTooLong(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	longSearch := strings.Repeat("a", 101)
	req := makeRequest("GET", "/conversations?search="+longSearch, "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestList_400_MissingUserID(t *testing.T) {
	h := NewConversationHandler(&mockConversationService{})
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations", "", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestList_500(t *testing.T) {
	mock := &mockConversationService{
		listErr: service.ErrConversationNotFound, // any error maps to 500 in List
	}
	h := NewConversationHandler(mock)
	r := newTestRouter(h)

	req := makeRequest("GET", "/conversations", "", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// Mock implementation - simple struct-based mock
// =============================================================================

type mockConversationService struct {
	createResult  service.ConversationDetailResponse
	createErr     error
	getByIDResult service.ConversationDetailResponse
	getByIDErr    error
	listResult    service.ListConversationsResult
	listErr       error
	updateResult  service.ConversationDetailResponse
	updateErr     error
	archiveResult service.ConversationResponse
	archiveErr    error
	addPartResult []service.ParticipantResponse
	addPartErr    error
	removePartErr error
}

func (m *mockConversationService) Create(_ context.Context, _ service.CreateConversationInput) (service.ConversationDetailResponse, error) {
	return m.createResult, m.createErr
}

func (m *mockConversationService) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (service.ConversationDetailResponse, error) {
	return m.getByIDResult, m.getByIDErr
}

func (m *mockConversationService) ListByUser(_ context.Context, _ service.ListConversationsInput) (service.ListConversationsResult, error) {
	return m.listResult, m.listErr
}

func (m *mockConversationService) Update(_ context.Context, _ service.UpdateConversationInput) (service.ConversationDetailResponse, error) {
	return m.updateResult, m.updateErr
}

func (m *mockConversationService) Archive(_ context.Context, _ uuid.UUID, _ uuid.UUID) (service.ConversationResponse, error) {
	return m.archiveResult, m.archiveErr
}

func (m *mockConversationService) AddParticipants(_ context.Context, _ service.AddParticipantsInput) ([]service.ParticipantResponse, error) {
	return m.addPartResult, m.addPartErr
}

func (m *mockConversationService) RemoveParticipant(_ context.Context, _ service.RemoveParticipantInput) error {
	return m.removePartErr
}
