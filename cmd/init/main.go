package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/renovatebot/maintainer-dashboard/internal/db"
)

// Retrieve the value of the -path flag argument

func main() {
	logger := slog.Default()

	path := flag.String("db", "dashboard.sqlite", "Path to the SQLite database file")
	flag.Parse()

	if path == nil {
		logger.Error("Missing -db parameter")
		os.Exit(1)
	}

	if _, err := os.Stat(*path); os.IsExist(err) {
		logger.Warn(fmt.Sprintf("Path '%v' already exists. Migrating", *path))
	} else if err != nil && !os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("Failed to check if file exists: %v", err), "err", err)
	}

	sqlDB, err := db.Open(*path)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open database at %s: %v", *path, err), "err", err)
		os.Exit(1)
	}

	err = db.ApplySchema(sqlDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to initialise/migrate database: %v", err), "err", err)
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("Successfully initialised/migrated database at %v", *path))
}
