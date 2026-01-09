package db

import (
	"database/sql"
	"net/url"

	_ "modernc.org/sqlite"
)

var defaultDBPragmasForWrite = map[string]string{
	"foreign_keys": "On",
}

func Open(path string) (*sql.DB, error) {
	path = constructDatabaseString(path, defaultDBPragmasForWrite)

	return sql.Open("sqlite", path)
}

// via https://gitlab.com/tanna.dev/dependency-management-data/-/blob/main/cmd/dmd/cmd/root.go#L205
func constructDatabaseString(databasePath string, pragmas map[string]string) string {
	u, err := url.Parse(databasePath)
	if err != nil {
		return ""
	}

	q := u.Query()
	for k, v := range pragmas {
		q.Add("_pragma", k+"="+v)
	}
	u.RawQuery = q.Encode()

	return u.String()
}
