package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	db "github.com/retich-corp/messaging/internal/db"
)

// =============================================================================
// Mock Store
// =============================================================================

type mockStore struct {
	// Conversation queries
	createConvResult db.Conversation
	createConvErr    error
	getConvResult    db.Conversation
	getConvErr       error
	updateConvResult db.Conversation
	updateConvErr    error
	archiveResult    db.Conversation
	archiveErr       error
	listConvResult   []db.Conversation
	listConvErr      error
	countConvResult  int64
	countConvErr     error
	searchConvResult []db.Conversation
	searchConvErr    error
	countSearchResult int64
	countSearchErr   error
	deleteConvErr    error
	unarchiveResult  db.Conversation
	unarchiveErr     error
	directConvResult db.Conversation
	directConvErr    error

	// Participant queries
	addPartResult    db.ConversationParticipant
	addPartErr       error
	removePartErr    error
	getPartResult    db.ConversationParticipant
	getPartErr       error
	listPartResult   []db.ConversationParticipant
	listPartErr      error
	countPartResult  int64
	countPartErr     error

	// Transaction
	beginTxResult *mockStoreTx
	beginTxErr    error
}

func (m *mockStore) BeginTx(_ context.Context) (db.StoreTx, error) {
	if m.beginTxErr != nil {
		return nil, m.beginTxErr
	}
	if m.beginTxResult != nil {
		return m.beginTxResult, nil
	}
	// Default: return a tx that mirrors the store
	return &mockStoreTx{store: m}, nil
}

func (m *mockStore) CreateConversation(_ context.Context, _ db.CreateConversationParams) (db.Conversation, error) {
	return m.createConvResult, m.createConvErr
}
func (m *mockStore) GetConversationByID(_ context.Context, _ uuid.UUID) (db.Conversation, error) {
	return m.getConvResult, m.getConvErr
}
func (m *mockStore) UpdateConversation(_ context.Context, _ db.UpdateConversationParams) (db.Conversation, error) {
	return m.updateConvResult, m.updateConvErr
}
func (m *mockStore) ArchiveConversation(_ context.Context, _ uuid.UUID) (db.Conversation, error) {
	return m.archiveResult, m.archiveErr
}
func (m *mockStore) UnarchiveConversation(_ context.Context, _ uuid.UUID) (db.Conversation, error) {
	return m.unarchiveResult, m.unarchiveErr
}
func (m *mockStore) DeleteConversation(_ context.Context, _ uuid.UUID) error {
	return m.deleteConvErr
}
func (m *mockStore) ListConversationsByUser(_ context.Context, _ db.ListConversationsByUserParams) ([]db.Conversation, error) {
	return m.listConvResult, m.listConvErr
}
func (m *mockStore) CountConversationsByUser(_ context.Context, _ uuid.UUID) (int64, error) {
	return m.countConvResult, m.countConvErr
}
func (m *mockStore) SearchConversationsByUser(_ context.Context, _ db.SearchConversationsByUserParams) ([]db.Conversation, error) {
	return m.searchConvResult, m.searchConvErr
}
func (m *mockStore) CountSearchConversationsByUser(_ context.Context, _ db.CountSearchConversationsByUserParams) (int64, error) {
	return m.countSearchResult, m.countSearchErr
}
func (m *mockStore) GetDirectConversationBetweenUsers(_ context.Context, _ db.GetDirectConversationBetweenUsersParams) (db.Conversation, error) {
	return m.directConvResult, m.directConvErr
}
func (m *mockStore) AddParticipant(_ context.Context, _ db.AddParticipantParams) (db.ConversationParticipant, error) {
	return m.addPartResult, m.addPartErr
}
func (m *mockStore) RemoveParticipant(_ context.Context, _ db.RemoveParticipantParams) error {
	return m.removePartErr
}
func (m *mockStore) GetParticipant(_ context.Context, _ db.GetParticipantParams) (db.ConversationParticipant, error) {
	return m.getPartResult, m.getPartErr
}
func (m *mockStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]db.ConversationParticipant, error) {
	return m.listPartResult, m.listPartErr
}
func (m *mockStore) CountActiveParticipants(_ context.Context, _ uuid.UUID) (int64, error) {
	return m.countPartResult, m.countPartErr
}

// mockStoreTx delegates to its parent store for simplicity.
type mockStoreTx struct {
	store       *mockStore
	commitErr   error
	rollbackErr error
}

func (t *mockStoreTx) Commit() error   { return t.commitErr }
func (t *mockStoreTx) Rollback() error { return t.rollbackErr }

func (t *mockStoreTx) CreateConversation(ctx context.Context, arg db.CreateConversationParams) (db.Conversation, error) {
	return t.store.CreateConversation(ctx, arg)
}
func (t *mockStoreTx) GetConversationByID(ctx context.Context, id uuid.UUID) (db.Conversation, error) {
	return t.store.GetConversationByID(ctx, id)
}
func (t *mockStoreTx) UpdateConversation(ctx context.Context, arg db.UpdateConversationParams) (db.Conversation, error) {
	return t.store.UpdateConversation(ctx, arg)
}
func (t *mockStoreTx) ArchiveConversation(ctx context.Context, id uuid.UUID) (db.Conversation, error) {
	return t.store.ArchiveConversation(ctx, id)
}
func (t *mockStoreTx) UnarchiveConversation(ctx context.Context, id uuid.UUID) (db.Conversation, error) {
	return t.store.UnarchiveConversation(ctx, id)
}
func (t *mockStoreTx) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	return t.store.DeleteConversation(ctx, id)
}
func (t *mockStoreTx) ListConversationsByUser(ctx context.Context, arg db.ListConversationsByUserParams) ([]db.Conversation, error) {
	return t.store.ListConversationsByUser(ctx, arg)
}
func (t *mockStoreTx) CountConversationsByUser(ctx context.Context, id uuid.UUID) (int64, error) {
	return t.store.CountConversationsByUser(ctx, id)
}
func (t *mockStoreTx) SearchConversationsByUser(ctx context.Context, arg db.SearchConversationsByUserParams) ([]db.Conversation, error) {
	return t.store.SearchConversationsByUser(ctx, arg)
}
func (t *mockStoreTx) CountSearchConversationsByUser(ctx context.Context, arg db.CountSearchConversationsByUserParams) (int64, error) {
	return t.store.CountSearchConversationsByUser(ctx, arg)
}
func (t *mockStoreTx) GetDirectConversationBetweenUsers(ctx context.Context, arg db.GetDirectConversationBetweenUsersParams) (db.Conversation, error) {
	return t.store.GetDirectConversationBetweenUsers(ctx, arg)
}
func (t *mockStoreTx) AddParticipant(ctx context.Context, arg db.AddParticipantParams) (db.ConversationParticipant, error) {
	return t.store.AddParticipant(ctx, arg)
}
func (t *mockStoreTx) RemoveParticipant(ctx context.Context, arg db.RemoveParticipantParams) error {
	return t.store.RemoveParticipant(ctx, arg)
}
func (t *mockStoreTx) GetParticipant(ctx context.Context, arg db.GetParticipantParams) (db.ConversationParticipant, error) {
	return t.store.GetParticipant(ctx, arg)
}
func (t *mockStoreTx) ListParticipants(ctx context.Context, id uuid.UUID) ([]db.ConversationParticipant, error) {
	return t.store.ListParticipants(ctx, id)
}
func (t *mockStoreTx) CountActiveParticipants(ctx context.Context, id uuid.UUID) (int64, error) {
	return t.store.CountActiveParticipants(ctx, id)
}

// =============================================================================
// Mock UserValidator
// =============================================================================

type mockUserValidator struct {
	invalidIDs []uuid.UUID
	err        error
}

func (m *mockUserValidator) ValidateUserIDs(_ context.Context, _ []uuid.UUID) ([]uuid.UUID, error) {
	return m.invalidIDs, m.err
}

// =============================================================================
// Helpers
// =============================================================================

func groupConversation() db.Conversation {
	return db.Conversation{
		ID:   uuid.New(),
		Type: db.ConversationTypeGroup,
		Name: sql.NullString{String: "Test Group", Valid: true},
	}
}

func directConversation() db.Conversation {
	return db.Conversation{
		ID:   uuid.New(),
		Type: db.ConversationTypeDirect,
	}
}

func participant(userID uuid.UUID, role string) db.ConversationParticipant {
	return db.ConversationParticipant{
		ID:     uuid.New(),
		UserID: userID,
		Role:   sql.NullString{String: role, Valid: true},
	}
}

func strPtr(s string) *string { return &s }

// =============================================================================
// Create tests
// =============================================================================

func TestCreate_Group_Success(t *testing.T) {
	conv := groupConversation()
	store := &mockStore{
		directConvErr:    sql.ErrNoRows,
		createConvResult: conv,
		addPartResult:    db.ConversationParticipant{ID: uuid.New()},
		listPartResult:   []db.ConversationParticipant{},
	}
	svc := NewConversationService(store, nil)

	result, err := svc.Create(context.Background(), CreateConversationInput{
		Type:           "group",
		Name:           strPtr("Test"),
		CreatorID:      uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "group" {
		t.Errorf("expected type group, got %s", result.Type)
	}
}

func TestCreate_Direct_Success(t *testing.T) {
	conv := directConversation()
	store := &mockStore{
		directConvErr:    sql.ErrNoRows,
		createConvResult: conv,
		addPartResult:    db.ConversationParticipant{ID: uuid.New()},
		listPartResult:   []db.ConversationParticipant{},
	}
	svc := NewConversationService(store, nil)

	result, err := svc.Create(context.Background(), CreateConversationInput{
		Type:           "direct",
		CreatorID:      uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "direct" {
		t.Errorf("expected type direct, got %s", result.Type)
	}
}

func TestCreate_DirectAlreadyExists(t *testing.T) {
	existingConv := directConversation()
	store := &mockStore{
		directConvResult: existingConv,
		listPartResult:   []db.ConversationParticipant{},
	}
	svc := NewConversationService(store, nil)

	_, err := svc.Create(context.Background(), CreateConversationInput{
		Type:           "direct",
		CreatorID:      uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})

	var existingErr *ExistingConversationError
	if !errors.As(err, &existingErr) {
		t.Fatalf("expected ExistingConversationError, got %v", err)
	}
}

func TestCreate_DirectRequiresOneParticipant(t *testing.T) {
	svc := NewConversationService(&mockStore{}, nil)

	_, err := svc.Create(context.Background(), CreateConversationInput{
		Type:           "direct",
		CreatorID:      uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New(), uuid.New()},
	})

	if !errors.Is(err, ErrDirectRequiresOneParticipant) {
		t.Fatalf("expected ErrDirectRequiresOneParticipant, got %v", err)
	}
}

func TestCreate_InvalidUserIDs(t *testing.T) {
	store := &mockStore{}
	validator := &mockUserValidator{invalidIDs: []uuid.UUID{uuid.New()}}
	svc := NewConversationService(store, validator)

	_, err := svc.Create(context.Background(), CreateConversationInput{
		Type:           "group",
		Name:           strPtr("Test"),
		CreatorID:      uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})

	if !errors.Is(err, ErrInvalidUserIDs) {
		t.Fatalf("expected ErrInvalidUserIDs, got %v", err)
	}
}

func TestCreate_UserValidationError(t *testing.T) {
	store := &mockStore{}
	validator := &mockUserValidator{err: errors.New("connection refused")}
	svc := NewConversationService(store, validator)

	_, err := svc.Create(context.Background(), CreateConversationInput{
		Type:           "group",
		Name:           strPtr("Test"),
		CreatorID:      uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestCreate_BeginTxError(t *testing.T) {
	store := &mockStore{
		directConvErr: sql.ErrNoRows,
		beginTxErr:    errors.New("tx error"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.Create(context.Background(), CreateConversationInput{
		Type:           "group",
		Name:           strPtr("Test"),
		CreatorID:      uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})

	if err == nil {
		t.Fatal("expected an error")
	}
}

// =============================================================================
// GetByID tests
// =============================================================================

func TestGetByID_Success(t *testing.T) {
	conv := groupConversation()
	userID := uuid.New()
	store := &mockStore{
		getConvResult:  conv,
		getPartResult:  participant(userID, "member"),
		listPartResult: []db.ConversationParticipant{participant(userID, "member")},
	}
	svc := NewConversationService(store, nil)

	result, err := svc.GetByID(context.Background(), conv.ID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Participants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(result.Participants))
	}
}

func TestGetByID_NotFound(t *testing.T) {
	store := &mockStore{
		getConvErr: sql.ErrNoRows,
	}
	svc := NewConversationService(store, nil)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestGetByID_NotParticipant(t *testing.T) {
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartErr:    sql.ErrNoRows,
	}
	svc := NewConversationService(store, nil)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotParticipant) {
		t.Fatalf("expected ErrNotParticipant, got %v", err)
	}
}

// =============================================================================
// ListByUser tests
// =============================================================================

func TestListByUser_Success(t *testing.T) {
	store := &mockStore{
		listConvResult:  []db.Conversation{groupConversation()},
		countConvResult: 1,
	}
	svc := NewConversationService(store, nil)

	result, err := svc.ListByUser(context.Background(), ListConversationsInput{
		UserID: uuid.New(),
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
}

func TestListByUser_WithSearch(t *testing.T) {
	search := "test"
	store := &mockStore{
		searchConvResult:  []db.Conversation{groupConversation()},
		countSearchResult: 1,
	}
	svc := NewConversationService(store, nil)

	result, err := svc.ListByUser(context.Background(), ListConversationsInput{
		UserID: uuid.New(),
		Limit:  20,
		Offset: 0,
		Search: &search,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
}

func TestListByUser_ListError(t *testing.T) {
	store := &mockStore{
		listConvErr: errors.New("db error"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.ListByUser(context.Background(), ListConversationsInput{UserID: uuid.New(), Limit: 20})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListByUser_CountError(t *testing.T) {
	store := &mockStore{
		listConvResult: []db.Conversation{},
		countConvErr:   errors.New("db error"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.ListByUser(context.Background(), ListConversationsInput{UserID: uuid.New(), Limit: 20})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListByUser_SearchError(t *testing.T) {
	search := "test"
	store := &mockStore{
		searchConvErr: errors.New("db error"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.ListByUser(context.Background(), ListConversationsInput{UserID: uuid.New(), Limit: 20, Search: &search})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListByUser_SearchCountError(t *testing.T) {
	search := "test"
	store := &mockStore{
		searchConvResult: []db.Conversation{},
		countSearchErr:   errors.New("db error"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.ListByUser(context.Background(), ListConversationsInput{UserID: uuid.New(), Limit: 20, Search: &search})
	if err == nil {
		t.Fatal("expected error")
	}
}

// =============================================================================
// Update tests
// =============================================================================

func TestUpdate_Success(t *testing.T) {
	conv := groupConversation()
	userID := uuid.New()
	store := &mockStore{
		getConvResult:    conv,
		getPartResult:    participant(userID, "owner"),
		updateConvResult: conv,
		listPartResult:   []db.ConversationParticipant{},
	}
	svc := NewConversationService(store, nil)

	name := "Updated"
	_, err := svc.Update(context.Background(), UpdateConversationInput{
		ConversationID: conv.ID,
		UserID:         userID,
		Name:           &name,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	store := &mockStore{getConvErr: sql.ErrNoRows}
	svc := NewConversationService(store, nil)

	_, err := svc.Update(context.Background(), UpdateConversationInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		Name:           strPtr("x"),
	})
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestUpdate_CannotModifyDirect(t *testing.T) {
	store := &mockStore{getConvResult: directConversation()}
	svc := NewConversationService(store, nil)

	_, err := svc.Update(context.Background(), UpdateConversationInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		Name:           strPtr("x"),
	})
	if !errors.Is(err, ErrCannotModifyDirect) {
		t.Fatalf("expected ErrCannotModifyDirect, got %v", err)
	}
}

func TestUpdate_NotParticipant(t *testing.T) {
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartErr:    sql.ErrNoRows,
	}
	svc := NewConversationService(store, nil)

	_, err := svc.Update(context.Background(), UpdateConversationInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		Name:           strPtr("x"),
	})
	if !errors.Is(err, ErrNotParticipant) {
		t.Fatalf("expected ErrNotParticipant, got %v", err)
	}
}

func TestUpdate_Forbidden_MemberRole(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartResult: participant(userID, "member"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.Update(context.Background(), UpdateConversationInput{
		ConversationID: uuid.New(),
		UserID:         userID,
		Name:           strPtr("x"),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// =============================================================================
// Archive tests
// =============================================================================

func TestArchive_Success(t *testing.T) {
	conv := groupConversation()
	userID := uuid.New()
	store := &mockStore{
		getConvResult: conv,
		getPartResult: participant(userID, "owner"),
		archiveResult: conv,
	}
	svc := NewConversationService(store, nil)

	_, err := svc.Archive(context.Background(), conv.ID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchive_NotFound(t *testing.T) {
	store := &mockStore{getConvErr: sql.ErrNoRows}
	svc := NewConversationService(store, nil)

	_, err := svc.Archive(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestArchive_NotParticipant(t *testing.T) {
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartErr:    sql.ErrNoRows,
	}
	svc := NewConversationService(store, nil)

	_, err := svc.Archive(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotParticipant) {
		t.Fatalf("expected ErrNotParticipant, got %v", err)
	}
}

func TestArchive_NotOwner(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartResult: participant(userID, "admin"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.Archive(context.Background(), uuid.New(), userID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// =============================================================================
// AddParticipants tests
// =============================================================================

func TestAddParticipants_Success(t *testing.T) {
	conv := groupConversation()
	userID := uuid.New()
	newPart := db.ConversationParticipant{ID: uuid.New(), UserID: uuid.New(), Role: sql.NullString{String: "member", Valid: true}}
	store := &mockStore{
		getConvResult: conv,
		getPartResult: participant(userID, "owner"),
		addPartResult: newPart,
	}
	svc := NewConversationService(store, nil)

	result, err := svc.AddParticipants(context.Background(), AddParticipantsInput{
		ConversationID: conv.ID,
		UserID:         userID,
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 added, got %d", len(result))
	}
}

func TestAddParticipants_Direct(t *testing.T) {
	store := &mockStore{getConvResult: directConversation()}
	svc := NewConversationService(store, nil)

	_, err := svc.AddParticipants(context.Background(), AddParticipantsInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if !errors.Is(err, ErrCannotModifyDirect) {
		t.Fatalf("expected ErrCannotModifyDirect, got %v", err)
	}
}

func TestAddParticipants_NotFound(t *testing.T) {
	store := &mockStore{getConvErr: sql.ErrNoRows}
	svc := NewConversationService(store, nil)

	_, err := svc.AddParticipants(context.Background(), AddParticipantsInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestAddParticipants_NotParticipant(t *testing.T) {
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartErr:    sql.ErrNoRows,
	}
	svc := NewConversationService(store, nil)

	_, err := svc.AddParticipants(context.Background(), AddParticipantsInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if !errors.Is(err, ErrNotParticipant) {
		t.Fatalf("expected ErrNotParticipant, got %v", err)
	}
}

func TestAddParticipants_Forbidden(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartResult: participant(userID, "member"),
	}
	svc := NewConversationService(store, nil)

	_, err := svc.AddParticipants(context.Background(), AddParticipantsInput{
		ConversationID: uuid.New(),
		UserID:         userID,
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAddParticipants_InvalidUserIDs(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartResult: participant(userID, "owner"),
	}
	validator := &mockUserValidator{invalidIDs: []uuid.UUID{uuid.New()}}
	svc := NewConversationService(store, validator)

	_, err := svc.AddParticipants(context.Background(), AddParticipantsInput{
		ConversationID: uuid.New(),
		UserID:         userID,
		ParticipantIDs: []uuid.UUID{uuid.New()},
	})
	if !errors.Is(err, ErrInvalidUserIDs) {
		t.Fatalf("expected ErrInvalidUserIDs, got %v", err)
	}
}

// =============================================================================
// RemoveParticipant tests
// =============================================================================

func TestRemoveParticipant_SelfLeave(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartResult: participant(userID, "member"),
	}
	svc := NewConversationService(store, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         userID,
		TargetUserID:   userID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveParticipant_OwnerRemovesMember(t *testing.T) {
	ownerID := uuid.New()
	memberID := uuid.New()

	callCount := 0
	store := &mockStore{
		getConvResult: groupConversation(),
	}
	// Override GetParticipant to return different results based on call order
	originalGetPart := store.getPartResult
	_ = originalGetPart

	// We need a more sophisticated mock for this test.
	// Use a custom store that tracks calls.
	customStore := &multiGetPartStore{
		mockStore: mockStore{getConvResult: groupConversation()},
		getPartResults: map[uuid.UUID]db.ConversationParticipant{
			ownerID:  participant(ownerID, "owner"),
			memberID: participant(memberID, "member"),
		},
	}
	svc := NewConversationService(customStore, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         ownerID,
		TargetUserID:   memberID,
	})
	_ = callCount
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveParticipant_NotFound(t *testing.T) {
	store := &mockStore{getConvErr: sql.ErrNoRows}
	svc := NewConversationService(store, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		TargetUserID:   uuid.New(),
	})
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestRemoveParticipant_DirectConversation(t *testing.T) {
	store := &mockStore{getConvResult: directConversation()}
	svc := NewConversationService(store, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         uuid.New(),
		TargetUserID:   uuid.New(), // different from UserID
	})
	if !errors.Is(err, ErrCannotModifyDirect) {
		t.Fatalf("expected ErrCannotModifyDirect, got %v", err)
	}
}

func TestRemoveParticipant_MemberCannotRemoveOthers(t *testing.T) {
	memberID := uuid.New()
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartResult: participant(memberID, "member"),
	}
	svc := NewConversationService(store, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         memberID,
		TargetUserID:   uuid.New(),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRemoveParticipant_CannotRemoveOwner(t *testing.T) {
	adminID := uuid.New()
	ownerID := uuid.New()
	customStore := &multiGetPartStore{
		mockStore: mockStore{getConvResult: groupConversation()},
		getPartResults: map[uuid.UUID]db.ConversationParticipant{
			adminID: participant(adminID, "admin"),
			ownerID: participant(ownerID, "owner"),
		},
	}
	svc := NewConversationService(customStore, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         adminID,
		TargetUserID:   ownerID,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRemoveParticipant_AdminCannotRemoveAdmin(t *testing.T) {
	admin1 := uuid.New()
	admin2 := uuid.New()
	customStore := &multiGetPartStore{
		mockStore: mockStore{getConvResult: groupConversation()},
		getPartResults: map[uuid.UUID]db.ConversationParticipant{
			admin1: participant(admin1, "admin"),
			admin2: participant(admin2, "admin"),
		},
	}
	svc := NewConversationService(customStore, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         admin1,
		TargetUserID:   admin2,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRemoveParticipant_SelfLeave_NotParticipant(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{
		getConvResult: groupConversation(),
		getPartErr:    sql.ErrNoRows,
	}
	svc := NewConversationService(store, nil)

	err := svc.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: uuid.New(),
		UserID:         userID,
		TargetUserID:   userID,
	})
	if !errors.Is(err, ErrNotParticipant) {
		t.Fatalf("expected ErrNotParticipant, got %v", err)
	}
}

// =============================================================================
// multiGetPartStore - returns different participants based on userID
// =============================================================================

type multiGetPartStore struct {
	mockStore
	getPartResults map[uuid.UUID]db.ConversationParticipant
}

func (s *multiGetPartStore) GetParticipant(_ context.Context, arg db.GetParticipantParams) (db.ConversationParticipant, error) {
	if p, ok := s.getPartResults[arg.UserID]; ok {
		return p, nil
	}
	return db.ConversationParticipant{}, sql.ErrNoRows
}
