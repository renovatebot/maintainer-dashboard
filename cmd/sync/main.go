package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/renovatebot/maintainer-dashboard/internal/db"
	"github.com/renovatebot/maintainer-dashboard/internal/github"
	"github.com/shurcooL/githubv4"
)

func main() {
	ctx := context.Background()
	logger := slog.Default()

	path := flag.String("db", "dashboard.sqlite", "Path to the SQLite database file")
	appId := flag.Int64("app-id", -1, "The App ID of the GitHub App to authenticate as")
	installationIds := flag.String("installation-ids", "", "Comma-separated list of installation IDs to rotate through based on rate limits (auto-discovers if not provided)")
	appKeyPath := flag.String("app-key", "", "Path to the GitHub App Private Key")

	flag.Parse()

	if path == nil {
		logger.Error("Missing -db parameter")
		os.Exit(1)
	}

	if appId == nil || *appId == -1 {
		logger.Error("Missing -app-id parameter")
		os.Exit(1)
	}

	if appKeyPath == nil {
		logger.Error("Missing -app-key parameter")
		os.Exit(1)
	}

	sqlDB, err := db.Open(*path)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open database at %s: %v", *path, err), "err", err)
		os.Exit(1)
	}
	queries := db.New(sqlDB)

	// Parse or discover installation IDs
	installationIdList, err := github.ParseOrDiscoverInstallations(ctx, *installationIds, *appId, *appKeyPath, logger)
	if err != nil {
		logger.Error("Failed to get installation IDs", "err", err)
		os.Exit(1)
	}

	clientPool := github.NewClientPool(*appId, installationIdList, *appKeyPath, logger)

	pw := progress.NewWriter()
	go pw.Render()

	lastDBUpdateVal, err := queries.FindMostRecentlyUpdatedDiscussion(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		lastDBUpdateVal = "1970-01-01T00:00:00Z"
	} else if err != nil {
		logger.Error(fmt.Sprintf("Failed to find most recently updated discussion in database: %v", err), "err", err)
		os.Exit(1)
	}

	lastDBUpdate, err := time.Parse(time.RFC3339, lastDBUpdateVal)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to parse last updated timestamp from database: %v", err), "err", err)
		os.Exit(1)
	}

	// Get a client for the initial check
	clientPair := clientPool.GetNextAvailableClient(ctx)
	lastUpdate, err := github.GetMostRecentlyUpdatedDiscussion(ctx, clientPair.RestClient, clientPair.GqlClient, "renovatebot", "renovate")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get most recently updated discussion from GitHub: %v", err), "err", err)
		os.Exit(1)
	}

	if lastDBUpdate.Equal(lastUpdate) || lastDBUpdate.After(lastUpdate) {
		logger.Info("No new discussions to sync, database is up-to-date", "lastUpdateOnGitHub", lastUpdate, "lastUpdateInDatabase", lastDBUpdate)
	} else {
		finished := false

		updateExistingDiscussionsTracker := progress.Tracker{
			Message: "Updating existing Discussions (and comments)",
		}
		pw.AppendTracker(&updateExistingDiscussionsTracker)

		var nextCursor *githubv4.String
		for !finished {
			// Get next available client with rate limit management
			clientPair := clientPool.GetNextAvailableClient(ctx)

			var discussions []github.MostRecentlyUpdatedDiscussion
			discussions, nextCursor, err = github.ListMostRecentlyUpdatedDiscussions(ctx, clientPair.RestClient, clientPair.GqlClient, "renovatebot", "renovate", 10, nextCursor)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to query discussion numbers: %v", err), "err", err)
				break
			}

			if discussions == nil {
				finished = true
				continue
			}

			if nextCursor == nil {
				finished = true
				continue
			}

			for _, discussion := range discussions {
				// TODO transaction
				if discussion.UpdatedAt.Before(lastDBUpdate) {
					finished = true
					break
				}

				// Get next available client for each discussion fetch
				clientPair = clientPool.GetNextAvailableClient(ctx)
				d, comments, err := github.RetrieveDiscussionAndComments(ctx, clientPair.RestClient, clientPair.GqlClient, "renovatebot", "renovate", discussion.Number)
				if err != nil {
					updateExistingDiscussionsTracker.IncrementWithError(1)
					logger.Error(fmt.Sprintf("Failed to retrieve discussion and comments for #%d: %v", discussion.Number, err), "err", err)
					continue
				}

				err = queries.InsertDiscussion(ctx, d)
				if err != nil {
					updateExistingDiscussionsTracker.IncrementWithError(1)
					logger.Error(fmt.Sprintf("Failed to insert discussion #%d: %v", discussion.Number, err), "err", err)
					continue
				}

				for _, comment := range comments {
					err = queries.InsertDiscussionComment(ctx, comment)
					if err != nil {
						updateExistingDiscussionsTracker.IncrementWithError(1)
						logger.Error(fmt.Sprintf("Failed to insert comment for discussion #%d: %v", discussion.Number, err), "err", err)
						continue
					}
				}

				updateExistingDiscussionsTracker.Increment(1)
			}
		}
		updateExistingDiscussionsTracker.MarkAsDone()
	}
}
