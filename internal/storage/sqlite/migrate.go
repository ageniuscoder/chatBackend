package sqlite

import (
	"os"
	"strings"
)

func (s *Sqlite) Migrate() error {
	b, err := os.ReadFile("sql/schema.sql")
	if err != nil {
		return err
	}
	stmts := strings.Split(string(b), ";\n")

	for _, stmt := range stmts {
		st := strings.TrimSpace(stmt)
		if st == "" {
			continue
		}
		if _, err = s.Db.Exec(st); err != nil {
			return err
		}
	}
	return nil
}
