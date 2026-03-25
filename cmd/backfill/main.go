package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/renovatebot/maintainer-dashboard/internal/db"
	"github.com/renovatebot/maintainer-dashboard/internal/github"
)

func main() {
	ctx := context.Background()
	logger := slog.Default()

	path := flag.String("db", "dashboard.sqlite", "Path to the SQLite database file")
	appId := flag.Int64("app-id", -1, "The App ID of the GitHub App to authenticate as")
	installationIds := flag.String("installation-ids", "", "Comma-separated list of installation IDs to rotate through based on rate limits (auto-discovers if not provided)")
	appKeyPath := flag.String("app-key", "", "Path to the GitHub App Private Key")
	fromNumber := flag.Int64("from", 0, "Only process discussions from this number onwards (inclusive)")

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

	discussionNumbers, err := queries.FindKnownDiscussions(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to find known discussions: %v", err), "err", err)
		os.Exit(1)
	}

	updateExistingDiscussionsTracker := progress.Tracker{
		Message: "Backfilling updates to existing Discussions (and comments)",
		Total:   int64(len(discussionNumbers)),
	}
	pw.AppendTracker(&updateExistingDiscussionsTracker)

	// Filter discussions if -from flag is provided
	if fromNumber != nil && *fromNumber > 0 {
		var filtered []int64
		for _, num := range discussionNumbers {
			if num >= *fromNumber {
				filtered = append(filtered, num)
			} else {
				updateExistingDiscussionsTracker.Increment(1)
			}
		}
		discussionNumbers = filtered
		logger.Info(fmt.Sprintf("Filtered to %d discussions from #%d onwards", len(discussionNumbers), *fromNumber))
	}

	for _, discussion := range discussionNumbers {
		// Get next available client with sufficient rate limits
		client := clientPool.GetNextAvailableClient(ctx)

		d, comments, err := github.RetrieveDiscussionAndComments(ctx, client.RestClient, client.GqlClient, "renovatebot", "renovate", discussion)
		if err != nil {
			updateExistingDiscussionsTracker.IncrementWithError(1)
			logger.Error(fmt.Sprintf("Failed to retrieve discussion and comments for #%d: %v", discussion, err), "err", err)
			continue
		}

		err = queries.InsertDiscussion(ctx, d)
		if err != nil {
			updateExistingDiscussionsTracker.IncrementWithError(1)
			logger.Error(fmt.Sprintf("Failed to insert discussion #%d: %v", discussion, err), "err", err)
			continue
		}

		for _, comment := range comments {
			err = queries.InsertDiscussionComment(ctx, comment)
			if err != nil {
				updateExistingDiscussionsTracker.IncrementWithError(1)
				logger.Error(fmt.Sprintf("Failed to insert comment for discussion #%d: %v", discussion, err), "err", err)
				continue
			}
		}

		updateExistingDiscussionsTracker.Increment(1)
	}
	updateExistingDiscussionsTracker.MarkAsDone()
}
