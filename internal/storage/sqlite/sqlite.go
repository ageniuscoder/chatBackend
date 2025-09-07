package sqlite

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

type Sqlite struct {
	Db *sql.DB
}

func New(dsn string) (*Sqlite, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, err
	}

	// Enable WAL for better concurrency
	_, _ = db.Exec(`PRAGMA journal_mode=WAL;`)

	return &Sqlite{
		Db: db,
	}, nil
}

func (s *Sqlite) Ping(ctx context.Context) error {
	return s.Db.PingContext(ctx)
}
