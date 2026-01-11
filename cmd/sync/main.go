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
	// TODO make this auto-detect installation(s)
	installationId := flag.Int64("installation-id", -1, "The installation ID of the GitHub App to authenticate with")
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

	client := github.NewClient(*appId, *installationId, *appKeyPath)
	gqlClient := github.NewGraphQLClient(*appId, *installationId, *appKeyPath)

	pw := progress.NewWriter()
	go pw.Render()

	lastDBUpdateVal, err := queries.FindMostRecentlyUpdatedDiscussion(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		lastDBUpdateVal = "1970-01-01T00:00:00Z"
	} else if err != nil {
		logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
		os.Exit(1)
	}

	lastDBUpdate, err := time.Parse(time.RFC3339, lastDBUpdateVal)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
		os.Exit(1)
	}

	lastUpdate, err := github.GetMostRecentlyUpdatedDiscussion(ctx, client, gqlClient, "renovatebot", "renovate")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
		os.Exit(1)
	}

	if lastDBUpdate.Equal(lastUpdate) || lastDBUpdate.After(lastUpdate) {
		logger.Info("**??**, but up-to-date", "lastUpdateOnGitHub", lastUpdate, "lastUpdateInDatabase", lastDBUpdate)
	} else {
		finished := false

		updateExistingDiscussionsTracker := progress.Tracker{
			Message: "Updating existing Discussions (and comments)",
		}
		pw.AppendTracker(&updateExistingDiscussionsTracker)

		var nextCursor *githubv4.String
		for !finished {
			var discussions []github.MostRecentlyUpdatedDiscussion
			discussions, nextCursor, err = github.ListMostRecentlyUpdatedDiscussions(ctx, client, gqlClient, "renovatebot", "renovate", 10, nextCursor)
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

				d, comments, err := github.RetrieveDiscussionAndComments(ctx, client, gqlClient, "renovatebot", "renovate", discussion.Number)
				if err != nil {
					updateExistingDiscussionsTracker.IncrementWithError(1)
					logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
					continue
				}

				err = queries.InsertDiscussion(ctx, d)
				if err != nil {
					updateExistingDiscussionsTracker.IncrementWithError(1)
					logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
					continue
				}

				for _, comment := range comments {
					err = queries.InsertDiscussionComment(ctx, comment)
					if err != nil {
						updateExistingDiscussionsTracker.IncrementWithError(1)
						logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
						continue
					}
				}

				updateExistingDiscussionsTracker.Increment(1)
			}
		}
		updateExistingDiscussionsTracker.MarkAsDone()
	}
}
