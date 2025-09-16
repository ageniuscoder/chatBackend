package postgres

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
)

type Postgres struct {
	Db *sql.DB
}

func New(dsn string) (*Postgres, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Postgres{
		Db: db,
	}, nil
}

func (s *Postgres) Ping(ctx context.Context) error {
	return s.Db.PingContext(ctx)
}
