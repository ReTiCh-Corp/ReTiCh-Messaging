package db

import (
	"testing"
)

// Compile-time check: SQLStore must satisfy Store interface.
var _ Store = (*SQLStore)(nil)

func TestNewSQLStore(t *testing.T) {
	// NewSQLStore with a nil db panics only when used, not on construction.
	// This verifies the constructor returns a non-nil store.
	store := NewSQLStore(nil)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}
