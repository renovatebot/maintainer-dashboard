package db

import (
	"database/sql"
	"strings"
)

// Migrate applies CreateTablesQuery to sqlDB. It is safe to run against a
// database that already has some or all of the schema applied: statements
// are executed individually, and an "add column" that has already been
// applied on a previous run is skipped rather than aborting the rest of the
// schema.
func Migrate(sqlDB *sql.DB) error {
	for _, statement := range strings.Split(CreateTablesQuery, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		if _, err := sqlDB.Exec(statement); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}

			return err
		}
	}

	return nil
}
