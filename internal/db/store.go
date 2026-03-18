package db

import (
	"context"
	"database/sql"
)

// Store wraps Querier with transaction support for testability.
type Store interface {
	Querier
	BeginTx(ctx context.Context) (StoreTx, error)
}

// StoreTx is a transactional store that can be committed or rolled back.
type StoreTx interface {
	Querier
	Commit() error
	Rollback() error
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

func (s *SQLStore) BeginTx(ctx context.Context) (StoreTx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlStoreTx{
		Queries: s.Queries.WithTx(tx),
		tx:      tx,
	}, nil
}

type sqlStoreTx struct {
	*Queries
	tx *sql.Tx
}

func (t *sqlStoreTx) Commit() error {
	return t.tx.Commit()
}

func (t *sqlStoreTx) Rollback() error {
	return t.tx.Rollback()
}
