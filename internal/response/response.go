package response

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Data       interface{}     `json:"data,omitempty"`
	Error      *APIError       `json:"error,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type PaginationMeta struct {
	Total  int64 `json:"total"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func JSON(w http.ResponseWriter, status int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, APIResponse{Data: data})
}

func SuccessWithPagination(w http.ResponseWriter, data interface{}, pagination PaginationMeta) {
	JSON(w, http.StatusOK, APIResponse{Data: data, Pagination: &pagination})
}

func ValidationError(w http.ResponseWriter, details map[string]string) {
	JSON(w, http.StatusBadRequest, APIResponse{
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid request parameters",
			Details: details,
		},
	})
}

func NotFound(w http.ResponseWriter, resource string) {
	JSON(w, http.StatusNotFound, APIResponse{
		Error: &APIError{
			Code:    "NOT_FOUND",
			Message: resource + " not found",
		},
	})
}

func InternalError(w http.ResponseWriter) {
	JSON(w, http.StatusInternalServerError, APIResponse{
		Error: &APIError{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
		},
	})
}

func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, APIResponse{
		Error: &APIError{
			Code:    "BAD_REQUEST",
			Message: message,
		},
	})
}

func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, APIResponse{
		Error: &APIError{
			Code:    "FORBIDDEN",
			Message: message,
		},
	})
}

func Conflict(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusConflict, APIResponse{
		Data: data,
		Error: &APIError{
			Code:    "CONFLICT",
			Message: message,
		},
	})
}
