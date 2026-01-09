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
	listDiscussionsTracker := progress.Tracker{
		Message: "List Discussions",
		Total:   0,
	}
	pw.AppendTracker(&listDiscussionsTracker)

	go pw.Render()

	done := make(chan struct{})
	results := make(chan github.ListDiscussionNumbersResult, 2)
	go github.ListDiscussionNumbers(ctx, client, gqlClient, "renovatebot", "renovate", results, done)

	var discussionsToCheck []int64

	finished := false
	for !finished {
		select {
		case res := <-results:
			if res.Error != nil {
				listDiscussionsTracker.IncrementWithError(1)
				logger.Error(fmt.Sprintf("Failed to query discussion numbers: %v", err), "err", err)
			} else {
				listDiscussionsTracker.Increment(1)
				discussionsToCheck = append(discussionsToCheck, res.Number)
			}
		case <-done:
			finished = true
		case <-ctx.Done():
			finished = true
		}
	}

	listDiscussionsTracker.MarkAsDone()

	// discussionsToSync, err := queries.FindMissingDiscussions(ctx, []any{discussionsToCheck})
	found, err := queries.FindKnownDiscussions(ctx, discussionsToCheck)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
		os.Exit(1)
	}

	discussionsToSync := missing(discussionsToCheck, found)
	fmt.Printf("discussionsToSync: %v\n", discussionsToSync)

	// TODO look up existing copies

	retrieveMissingDiscussionsTracker := progress.Tracker{
		Message: "Retrieving missing Discussions (and comments)",
		Total:   int64(len(discussionsToSync)),
	}
	pw.AppendTracker(&retrieveMissingDiscussionsTracker)

	for _, number := range discussionsToSync {
		d, _, err := github.RetrieveDiscussionAndComments(ctx, client, gqlClient, "renovatebot", "renovate", number)
		if err != nil {
			retrieveMissingDiscussionsTracker.IncrementWithError(1)
			logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
			continue
		}

		err = queries.InsertDiscussion(ctx, d)
		if err != nil {
			retrieveMissingDiscussionsTracker.IncrementWithError(1)
			logger.Error(fmt.Sprintf("Failed to query **??**: %v", err), "err", err)
			continue
		}
		retrieveMissingDiscussionsTracker.Increment(1)
	}

	retrieveMissingDiscussionsTracker.MarkAsDone()

	// fmt.Printf("err: %v\n", err)
	// b, _ := json.Marshal(d)
	// fmt.Printf("b: %s\n", b)
	//
	// panic("todo")
	//
}

func missing(numbers, existing []int64) []int64 {
	existingSet := make(map[int64]struct{})
	for _, n := range existing {
		existingSet[n] = struct{}{}
	}
	var missing []int64
	for _, n := range numbers {
		if _, ok := existingSet[n]; !ok {
			missing = append(missing, n)
		}
	}
	return missing
}
