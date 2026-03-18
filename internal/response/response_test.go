package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	resp := APIResponse{Data: map[string]string{"key": "value"}}

	JSON(w, http.StatusOK, resp)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	data, ok := got.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if data["key"] != "value" {
		t.Errorf("expected key=value, got %v", data["key"])
	}
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	Success(w, http.StatusCreated, "created")

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Data != "created" {
		t.Errorf("expected data=created, got %v", got.Data)
	}
	if got.Error != nil {
		t.Error("expected no error")
	}
}

func TestSuccessWithPagination(t *testing.T) {
	w := httptest.NewRecorder()
	pagination := PaginationMeta{Total: 100, Limit: 10, Offset: 0}
	SuccessWithPagination(w, []string{"a", "b"}, pagination)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Pagination == nil {
		t.Fatal("expected pagination to be set")
	}
	if got.Pagination.Total != 100 {
		t.Errorf("expected total=100, got %d", got.Pagination.Total)
	}
	if got.Pagination.Limit != 10 {
		t.Errorf("expected limit=10, got %d", got.Pagination.Limit)
	}
	if got.Pagination.Offset != 0 {
		t.Errorf("expected offset=0, got %d", got.Pagination.Offset)
	}
}

func TestValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]string{"name": "required"}
	ValidationError(w, details)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Error == nil {
		t.Fatal("expected error to be set")
	}
	if got.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code=VALIDATION_ERROR, got %s", got.Error.Code)
	}
	if got.Error.Details["name"] != "required" {
		t.Errorf("expected details name=required, got %v", got.Error.Details)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "conversation")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Error == nil {
		t.Fatal("expected error to be set")
	}
	if got.Error.Code != "NOT_FOUND" {
		t.Errorf("expected code=NOT_FOUND, got %s", got.Error.Code)
	}
	if got.Error.Message != "conversation not found" {
		t.Errorf("expected message 'conversation not found', got %s", got.Error.Message)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Error == nil {
		t.Fatal("expected error to be set")
	}
	if got.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code=INTERNAL_ERROR, got %s", got.Error.Code)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Error == nil {
		t.Fatal("expected error to be set")
	}
	if got.Error.Code != "BAD_REQUEST" {
		t.Errorf("expected code=BAD_REQUEST, got %s", got.Error.Code)
	}
	if got.Error.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %s", got.Error.Message)
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Error == nil {
		t.Fatal("expected error to be set")
	}
	if got.Error.Code != "FORBIDDEN" {
		t.Errorf("expected code=FORBIDDEN, got %s", got.Error.Code)
	}
	if got.Error.Message != "access denied" {
		t.Errorf("expected message 'access denied', got %s", got.Error.Message)
	}
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	Conflict(w, "already exists", map[string]string{"id": "123"})

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var got APIResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Error == nil {
		t.Fatal("expected error to be set")
	}
	if got.Error.Code != "CONFLICT" {
		t.Errorf("expected code=CONFLICT, got %s", got.Error.Code)
	}
	if got.Error.Message != "already exists" {
		t.Errorf("expected message 'already exists', got %s", got.Error.Message)
	}
	if got.Data == nil {
		t.Error("expected data to be set")
	}
}
