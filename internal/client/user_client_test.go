package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestUserExists_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	exists, err := c.UserExists(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected user to exist")
	}
}

func TestUserExists_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	exists, err := c.UserExists(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected user to not exist")
	}
}

func TestUserExists_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	_, err := c.UserExists(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestUserExists_ConnectionError(t *testing.T) {
	c := NewUserClient("http://localhost:1") // nothing listening
	_, err := c.UserExists(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidateUserIDs_AllValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	invalidIDs, err := c.ValidateUserIDs(context.Background(), []uuid.UUID{uuid.New(), uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invalidIDs) != 0 {
		t.Errorf("expected no invalid IDs, got %d", len(invalidIDs))
	}
}

func TestValidateUserIDs_SomeInvalid(t *testing.T) {
	validID := uuid.New()
	invalidID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/"+validID.String() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	result, err := c.ValidateUserIDs(context.Background(), []uuid.UUID{validID, invalidID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 invalid ID, got %d", len(result))
	}
	if result[0] != invalidID {
		t.Errorf("expected invalid ID %s, got %s", invalidID, result[0])
	}
}

func TestValidateUserIDs_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	_, err := c.ValidateUserIDs(context.Background(), []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidateUserIDs_Empty(t *testing.T) {
	c := NewUserClient("http://localhost:1")
	invalidIDs, err := c.ValidateUserIDs(context.Background(), []uuid.UUID{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invalidIDs) != 0 {
		t.Errorf("expected no invalid IDs, got %d", len(invalidIDs))
	}
}
