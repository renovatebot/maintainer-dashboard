package db

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const maxBusyRetry = 5

// RetryOnBusy runs fn, retrying with backoff if it fails because the
// database is locked (SQLITE_BUSY/SQLITE_LOCKED), e.g. from a concurrent
// writer. This is on top of the busy_timeout pragma set in Open, which
// handles short-lived contention within a single call.
func RetryOnBusy(ctx context.Context, logger *slog.Logger, fn func() error) error {
	var err error
	for attempt := 0; attempt <= maxBusyRetry; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if !isBusyError(err) || attempt == maxBusyRetry {
			return err
		}

		backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
		logger.Warn("Database is locked, retrying", "attempt", attempt+1, "backoff", backoff, "err", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return err
}

func isBusyError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}

	return sqliteErr.Code() == sqlite3.SQLITE_BUSY || sqliteErr.Code() == sqlite3.SQLITE_LOCKED
}
