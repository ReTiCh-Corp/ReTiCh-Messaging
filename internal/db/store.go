package db

import (
	"context"
	"database/sql"
)

// Store wraps Querier with transaction support for testability.
type Store interface {
	Querier
	ExecTx(ctx context.Context, fn func(Querier) error) error
}

// SQLStore implements Store using a real SQL database.
type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewSQLStore(conn *sql.DB) *SQLStore {
	return &SQLStore{
		Queries: New(conn),
		db:      conn,
	}
}

// ExecTx executes fn within a database transaction. If fn returns an error,
// the transaction is rolled back. Otherwise, the transaction is committed.
func (s *SQLStore) ExecTx(ctx context.Context, fn func(Querier) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := s.Queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}

	return tx.Commit()
}
