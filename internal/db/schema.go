package db

import (
	"database/sql"
	_ "embed"
	"strings"
)

//go:embed schema.sql
var CreateTablesQuery string

// ApplySchema executes the schema SQL statement-by-statement against db.
// ALTER TABLE statements that fail because the column already exists are
// silently ignored, making this safe to run against an already-migrated DB.
func ApplySchema(sqlDB *sql.DB) error {
	for _, stmt := range strings.Split(CreateTablesQuery, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err := sqlDB.Exec(stmt)
		if err != nil && strings.Contains(err.Error(), "duplicate column name") {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}
