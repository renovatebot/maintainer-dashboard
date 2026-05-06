package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v81/github"
	"github.com/renovatebot/maintainer-dashboard/internal/db"
	"github.com/shurcooL/githubv4"
)

// ClientPair holds a REST and GraphQL client for a specific installation
type ClientPair struct {
	RestClient *github.Client
	GqlClient  *githubv4.Client
	InstallID  int64
}

// ClientPool manages multiple GitHub client pairs for rate limit rotation
type ClientPool struct {
	clients      []ClientPair
	currentIndex int
	logger       *slog.Logger
}

func NewClient(appId, installationId int64, appKeyPath string) *github.Client {
	// Shared transport to reuse TCP connections.
	tr := http.DefaultTransport

	// Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
	itr, err := ghinstallation.NewKeyFromFile(tr, appId, int64(installationId), appKeyPath)
	if err != nil {
		// TODO
		log.Fatal(err)
	}

	// Use installation transport with github.com/google/go-github
	return github.NewClient(&http.Client{Transport: itr})
}

func NewGraphQLClient(appId, installationId int64, appKeyPath string) *githubv4.Client {
	// Shared transport to reuse TCP connections.
	tr := http.DefaultTransport

	// Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
	itr, err := ghinstallation.NewKeyFromFile(tr, appId, int64(installationId), appKeyPath)
	if err != nil {
		// TODO
		log.Fatal(err)
	}

	return githubv4.NewClient(&http.Client{Transport: itr})
}

// ListInstallations retrieves all installations for a GitHub App
func ListInstallations(ctx context.Context, appId int64, appKeyPath string) ([]int64, error) {
	tr := http.DefaultTransport

	// Create App-level authentication (not installation-specific)
	atr, err := ghinstallation.NewAppsTransportKeyFromFile(tr, appId, appKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create app transport: %w", err)
	}

	client := github.NewClient(&http.Client{Transport: atr})

	// List all installations for this App
	installations, _, err := client.Apps.ListInstallations(ctx, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}

	var installationIds []int64
	for _, installation := range installations {
		if installation.ID != nil {
			installationIds = append(installationIds, *installation.ID)
		}
	}

	return installationIds, nil
}

// ParseOrDiscoverInstallations parses comma-separated installation IDs from a string,
// or auto-discovers them if the string is empty
func ParseOrDiscoverInstallations(ctx context.Context, installationIdsStr string, appId int64, appKeyPath string, logger *slog.Logger) ([]int64, error) {
	if installationIdsStr != "" {
		// Parse provided installation IDs
		return ParseInstallationIds(installationIdsStr)
	}

	// Auto-discover installations
	logger.Info("No installation IDs provided, auto-discovering...")
	discoveredIds, err := ListInstallations(ctx, appId, appKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover installations: %w", err)
	}
	if len(discoveredIds) == 0 {
		return nil, fmt.Errorf("no installations found for this GitHub App")
	}
	logger.Info(fmt.Sprintf("Discovered %d installation(s)", len(discoveredIds)), "installation_ids", discoveredIds)
	return discoveredIds, nil
}

// ParseInstallationIds parses a comma-separated string of installation IDs
func ParseInstallationIds(installationIdsStr string) ([]int64, error) {
	installationIdStrs := strings.Split(installationIdsStr, ",")
	var installationIdList []int64
	for _, idStr := range installationIdStrs {
		idStr = strings.TrimSpace(idStr)
		installID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid installation ID '%s': %w", idStr, err)
		}
		installationIdList = append(installationIdList, installID)
	}
	return installationIdList, nil
}

// NewClientPool creates a pool of GitHub clients from installation IDs
func NewClientPool(appId int64, installationIds []int64, appKeyPath string, logger *slog.Logger) *ClientPool {
	var clients []ClientPair
	for _, installID := range installationIds {
		restClient := NewClient(appId, installID, appKeyPath)
		gqlClient := NewGraphQLClient(appId, installID, appKeyPath)
		clients = append(clients, ClientPair{
			RestClient: restClient,
			GqlClient:  gqlClient,
			InstallID:  installID,
		})
	}
	return &ClientPool{
		clients:      clients,
		currentIndex: 0,
		logger:       logger,
	}
}

// GetNextAvailableClient returns a client pair with sufficient rate limits
// It checks both Core and GraphQL rate limits and rotates through clients if needed
// If all clients are rate limited, it waits until the earliest reset time
func (cp *ClientPool) GetNextAvailableClient(ctx context.Context) ClientPair {
	if len(cp.clients) == 0 {
		cp.logger.Error("No clients available in pool")
		return ClientPair{}
	}

	// Check current client's rate limits
	currentClient := cp.clients[cp.currentIndex]
	rateLimit, _, err := currentClient.RestClient.RateLimit.Get(ctx)
	if err != nil {
		cp.logger.Warn("Failed to check rate limit, using current client anyway", "err", err)
		return currentClient
	}

	if rateLimit == nil {
		return currentClient
	}

	// Check both Core (REST) and GraphQL rate limits
	coreRemaining := 0
	graphqlRemaining := 0
	var earliestResetNeeded time.Time
	needRotation := false

	if rateLimit.Core != nil {
		coreRemaining = rateLimit.Core.Remaining
		if coreRemaining < 100 {
			needRotation = true
			earliestResetNeeded = rateLimit.Core.Reset.Time
		}
	}

	if rateLimit.GraphQL != nil {
		graphqlRemaining = rateLimit.GraphQL.Remaining
		if graphqlRemaining < 100 {
			needRotation = true
			if earliestResetNeeded.IsZero() || rateLimit.GraphQL.Reset.Before(earliestResetNeeded) {
				earliestResetNeeded = rateLimit.GraphQL.Reset.Time
			}
		}
	}

	if !needRotation {
		return currentClient
	}

	// Need to rotate - log and move to next client
	cp.logger.Info(fmt.Sprintf("Rate limit low (Core: %d, GraphQL: %d remaining) for installation %d, rotating to next client. Reset at %v",
		coreRemaining, graphqlRemaining, currentClient.InstallID, earliestResetNeeded))
	cp.currentIndex = (cp.currentIndex + 1) % len(cp.clients)
	currentClient = cp.clients[cp.currentIndex]

	// Check if all clients are rate limited (either Core or GraphQL)
	allLimited := true
	var earliestReset time.Time
	for i, c := range cp.clients {
		rl, _, err := c.RestClient.RateLimit.Get(ctx)
		if err == nil && rl != nil {
			// Client is available if both Core and GraphQL have sufficient limits
			coreOk := rl.Core != nil && rl.Core.Remaining >= 100
			graphqlOk := rl.GraphQL != nil && rl.GraphQL.Remaining >= 100

			if coreOk && graphqlOk {
				allLimited = false
				cp.currentIndex = i
				currentClient = cp.clients[i]
				break
			}

			// Track earliest reset time across both rate limit types
			if rl.Core != nil && rl.Core.Remaining < 100 {
				if earliestReset.IsZero() || rl.Core.Reset.Before(earliestReset) {
					earliestReset = rl.Core.Reset.Time
				}
			}
			if rl.GraphQL != nil && rl.GraphQL.Remaining < 100 {
				if earliestReset.IsZero() || rl.GraphQL.Reset.Before(earliestReset) {
					earliestReset = rl.GraphQL.Reset.Time
				}
			}
		}
	}

	if allLimited {
		waitDuration := time.Until(earliestReset) + time.Second
		cp.logger.Warn(fmt.Sprintf("All clients rate limited (Core and/or GraphQL). Waiting until %v (%v)", earliestReset, waitDuration))
		time.Sleep(waitDuration)
	}

	return currentClient
}

func RetrieveDiscussionAndComments(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, org, repo string, number int64) (db.InsertDiscussionParams, []db.InsertDiscussionCommentParams, error) {
	var discussionQuery struct {
		Repository struct {
			Discussion struct {
				Title       githubv4.String
				URL         githubv4.URI
				StateReason *githubv4.String
				CreatedAt   githubv4.DateTime
				UpdatedAt   githubv4.DateTime
				ClosedAt    *githubv4.DateTime
				Author      struct {
					Login githubv4.String
				}
				Category struct {
					Name githubv4.String
				}
				Labels struct {
					Edges []struct {
						Node struct {
							Name githubv4.String
						}
					}
				} `graphql:"labels(first:100)"`
				AnswerChosenAt *githubv4.DateTime
				Answer         *struct {
					Author struct {
						Login githubv4.String
					}
				}
				Body           githubv4.String
				UpvoteCount    githubv4.Int
				AnswerChosenBy *struct {
					Login githubv4.String
				}
			} `graphql:"discussion(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]any{
		"owner":  githubv4.String(org),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(number),
	}

	err := gqlClient.Query(ctx, &discussionQuery, variables)
	if err != nil {
		return db.InsertDiscussionParams{}, nil, fmt.Errorf("failed to query %v/%v Discussion %v: %w", org, repo, number, err)
	}

	discussion := db.InsertDiscussionParams{
		Number: number,
		Title:  string(discussionQuery.Repository.Discussion.Title),
		Url:    discussionQuery.Repository.Discussion.URL.String(),
		// Updated below
		State:     "OPEN",
		CreatedAt: discussionQuery.Repository.Discussion.CreatedAt.Format(time.RFC3339),
		UpdatedAt: discussionQuery.Repository.Discussion.UpdatedAt.Format(time.RFC3339),
		// ClosedAt is handled below
		Author:       string(discussionQuery.Repository.Discussion.Author.Login),
		CategoryName: string(discussionQuery.Repository.Discussion.Category.Name),
		// AnswerChosenAt is handled below
		// AnswerBy is handled below
		Labels: []byte("[]"),
		Body: sql.NullString{
			String: string(discussionQuery.Repository.Discussion.Body),
			Valid:  true,
		},
		UpvoteCount: sql.NullInt64{
			Int64: int64(discussionQuery.Repository.Discussion.UpvoteCount),
			Valid: true,
		},
		// AnswerChosenBy is handled below
	}

	if discussionQuery.Repository.Discussion.StateReason != nil {
		discussion.State = string(*discussionQuery.Repository.Discussion.StateReason)
	}

	if discussionQuery.Repository.Discussion.ClosedAt != nil {
		discussion.ClosedAt = sql.NullString{
			String: discussionQuery.Repository.Discussion.ClosedAt.Format(time.RFC3339),
			Valid:  true,
		}
	}

	if discussionQuery.Repository.Discussion.AnswerChosenAt != nil {
		discussion.AnswerChosenAt = sql.NullString{
			String: discussionQuery.Repository.Discussion.AnswerChosenAt.Format(time.RFC3339),
			Valid:  true,
		}
	}

	if discussionQuery.Repository.Discussion.Answer != nil {
		discussion.AnsweredBy = sql.NullString{
			String: string(discussionQuery.Repository.Discussion.Answer.Author.Login),
			Valid:  true,
		}
	}

	if discussionQuery.Repository.Discussion.AnswerChosenBy != nil {
		discussion.AnswerChosenBy = sql.NullString{
			String: string(discussionQuery.Repository.Discussion.AnswerChosenBy.Login),
			Valid:  true,
		}
	}

	var labels []string
	for _, edge := range discussionQuery.Repository.Discussion.Labels.Edges {
		labels = append(labels, string(edge.Node.Name))
	}

	if len(labels) > 0 {
		discussion.Labels, err = json.Marshal(labels)
		if err != nil {
			return db.InsertDiscussionParams{}, nil, fmt.Errorf("failed to query: %w", err)
		}
	}

	var discussionCommentsQuery struct {
		Repository struct {
			Discussion struct {
				Comments struct {
					PageInfo struct {
						HasNextPage githubv4.Boolean
						EndCursor   githubv4.String
					}
					Nodes []struct {
						ID        string
						CreatedAt githubv4.DateTime
						UpdatedAt githubv4.DateTime
						Author    struct {
							Typename string `graphql:"__typename"`
							Login    githubv4.String
						}
						Body        githubv4.String
						UpvoteCount githubv4.Int
						Replies     struct {
							PageInfo struct {
								// TODO WARN
								HasNextPage githubv4.Boolean
							}
							Nodes []struct {
								ID        string
								CreatedAt githubv4.DateTime
								UpdatedAt githubv4.DateTime
								Author    struct {
									Typename string `graphql:"__typename"`
									Login    githubv4.String
								}
								ReplyTo struct {
									ID string
									// ID githubv4.ID
								}
								Body        githubv4.String
								UpvoteCount githubv4.Int
							}
						} `graphql:"replies(last: 100)"`
					}
				} `graphql:"comments(first: 100, after: $cursor)"`
			} `graphql:"discussion(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	var comments []db.InsertDiscussionCommentParams

	variables = map[string]any{
		"owner":  githubv4.String(org),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(number),
		"cursor": (*githubv4.String)(nil), // Null after argument to get first page.
	}

	for {
		err := gqlClient.Query(ctx, &discussionCommentsQuery, variables)
		if err != nil {
			return db.InsertDiscussionParams{}, nil, fmt.Errorf("failed to query: %w", err)
		}

		for _, node := range discussionCommentsQuery.Repository.Discussion.Comments.Nodes {
			author := string(node.Author.Login)
			if node.Author.Typename == "Bot" {
				author = author + "[bot]"
			}

			comments = append(comments, db.InsertDiscussionCommentParams{
				DiscussionNumber: number,
				ID:               node.ID,
				CreatedAt:        node.CreatedAt.Format(time.RFC3339),
				UpdatedAt:        node.UpdatedAt.Format(time.RFC3339),
				Author:           author,
				// top-level comments don't have a reply
				Body: sql.NullString{
					String: string(node.Body),
					Valid:  true,
				},
				UpvoteCount: sql.NullInt64{
					Int64: int64(node.UpvoteCount),
					Valid: true,
				},
			})

			if node.Replies.PageInfo.HasNextPage {
				slog.Warn(fmt.Sprintf("TODO: The %s/%s Discussion %s has a reply (ID %v) which has >100 replies. Only fetching last 100", org, repo, number, node.ID))
			}

			for _, reply := range node.Replies.Nodes {
				author := string(reply.Author.Login)
				if reply.Author.Typename == "Bot" {
					author = author + "[bot]"
				}

				comments = append(comments, db.InsertDiscussionCommentParams{
					DiscussionNumber: number,
					ID:               reply.ID,
					CreatedAt:        reply.CreatedAt.Format(time.RFC3339),
					UpdatedAt:        reply.UpdatedAt.Format(time.RFC3339),
					Author:           author,
					ReplyTo: sql.NullString{
						String: reply.ReplyTo.ID,
						Valid:  true,
					},
					Body: sql.NullString{
						String: string(reply.Body),
						Valid:  true,
					},
					UpvoteCount: sql.NullInt64{
						Int64: int64(reply.UpvoteCount),
						Valid: true,
					},
				})
			}
		}

		if !discussionCommentsQuery.Repository.Discussion.Comments.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(discussionCommentsQuery.Repository.Discussion.Comments.PageInfo.EndCursor)
	}

	return discussion, comments, nil
}

func GetMostRecentlyUpdatedDiscussion(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, org, repo string) (time.Time, error) {
	var q struct {
		Repository struct {
			Discussions struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
				Edges []struct {
					Node struct {
						Number    int64
						UpdatedAt githubv4.DateTime
					}
				}
			} `graphql:"discussions(first: 1, orderBy:{field:UPDATED_AT, direction:DESC})"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]any{
		"owner": githubv4.String(org),
		"name":  githubv4.String(repo),
	}

	err := gqlClient.Query(ctx, &q, variables)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query most recently updated discussion for %v/%v: %w", org, repo, err)
	}

	if len(q.Repository.Discussions.Edges) == 0 {
		return time.Time{}, nil
	}

	return q.Repository.Discussions.Edges[0].Node.UpdatedAt.Time, nil
}

type MostRecentlyUpdatedDiscussion struct {
	Number    int64
	UpdatedAt time.Time
}

func ListMostRecentlyUpdatedDiscussions(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, org, repo string, mostRecent int, cursor *githubv4.String) ([]MostRecentlyUpdatedDiscussion, *githubv4.String, error) {
	var q struct {
		Repository struct {
			Discussions struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
				Edges []struct {
					Node struct {
						Number    int64
						UpdatedAt githubv4.DateTime
					}
				}
			} `graphql:"discussions(first: $mostRecent, orderBy:{field:UPDATED_AT, direction:DESC}, after: $cursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]any{
		"owner":      githubv4.String(org),
		"name":       githubv4.String(repo),
		"mostRecent": githubv4.Int(mostRecent),
		"cursor":     cursor,
	}

	var results []MostRecentlyUpdatedDiscussion

	err := gqlClient.Query(ctx, &q, variables)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list most recently updated discussions for %v/%v: %w", org, repo, err)
	}
	for _, edge := range q.Repository.Discussions.Edges {
		results = append(results, MostRecentlyUpdatedDiscussion{
			Number:    edge.Node.Number,
			UpdatedAt: edge.Node.UpdatedAt.Time,
		})
	}

	return results, &q.Repository.Discussions.PageInfo.EndCursor, nil
}

func GetMostRecentlyUpdatedIssue(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, org, repo string) (time.Time, error) {
	var q struct {
		Repository struct {
			Issues struct {
				Edges []struct {
					Node struct {
						Number    int64
						UpdatedAt githubv4.DateTime
					}
				}
			} `graphql:"issues(first: 1, orderBy:{field:UPDATED_AT, direction:DESC})"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]any{
		"owner": githubv4.String(org),
		"name":  githubv4.String(repo),
	}

	err := gqlClient.Query(ctx, &q, variables)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query most recently updated issue for %v/%v: %w", org, repo, err)
	}

	if len(q.Repository.Issues.Edges) == 0 {
		return time.Time{}, nil
	}

	return q.Repository.Issues.Edges[0].Node.UpdatedAt.Time, nil
}

type MostRecentlyUpdatedIssue struct {
	Number    int64
	UpdatedAt time.Time
}

func ListMostRecentlyUpdatedIssues(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, org, repo string, mostRecent int, cursor *githubv4.String) ([]MostRecentlyUpdatedIssue, *githubv4.String, error) {
	var q struct {
		Repository struct {
			Issues struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
				Edges []struct {
					Node struct {
						Number    int64
						UpdatedAt githubv4.DateTime
					}
				}
			} `graphql:"issues(first: $mostRecent, orderBy:{field:UPDATED_AT, direction:DESC}, after: $cursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]any{
		"owner":      githubv4.String(org),
		"name":       githubv4.String(repo),
		"mostRecent": githubv4.Int(mostRecent),
		"cursor":     cursor,
	}

	var results []MostRecentlyUpdatedIssue

	err := gqlClient.Query(ctx, &q, variables)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list most recently updated issues for %v/%v: %w", org, repo, err)
	}
	for _, edge := range q.Repository.Issues.Edges {
		results = append(results, MostRecentlyUpdatedIssue{
			Number:    edge.Node.Number,
			UpdatedAt: edge.Node.UpdatedAt.Time,
		})
	}

	return results, &q.Repository.Issues.PageInfo.EndCursor, nil
}

func RetrieveIssueAndComments(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, org, repo string, number int64) (db.InsertIssueParams, []db.InsertIssueCommentParams, error) {
	var issueQuery struct {
		Repository struct {
			Issue struct {
				Title       githubv4.String
				URL         githubv4.URI
				State       githubv4.String
				StateReason *githubv4.String
				CreatedAt   githubv4.DateTime
				UpdatedAt   githubv4.DateTime
				ClosedAt    *githubv4.DateTime
				Author      *struct {
					Typename string `graphql:"__typename"`
					Login    githubv4.String
				}
				Labels struct {
					Edges []struct {
						Node struct {
							Name githubv4.String
						}
					}
				} `graphql:"labels(first:100)"`
				Body           githubv4.String
				Locked         githubv4.Boolean
				ReactionGroups []struct {
					Content  githubv4.String
					Reactors struct {
						TotalCount githubv4.Int
					}
				}
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]any{
		"owner":  githubv4.String(org),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(number),
	}

	err := gqlClient.Query(ctx, &issueQuery, variables)
	if err != nil {
		return db.InsertIssueParams{}, nil, fmt.Errorf("failed to query %v/%v Issue %v: %w", org, repo, number, err)
	}

	author := ""
	if issueQuery.Repository.Issue.Author != nil {
		author = string(issueQuery.Repository.Issue.Author.Login)
		if issueQuery.Repository.Issue.Author.Typename == "Bot" {
			author = author + "[bot]"
		}
	}

	locked := int64(0)
	if bool(issueQuery.Repository.Issue.Locked) {
		locked = 1
	}

	issue := db.InsertIssueParams{
		Number:    number,
		Title:     string(issueQuery.Repository.Issue.Title),
		Url:       issueQuery.Repository.Issue.URL.String(),
		State:     string(issueQuery.Repository.Issue.State),
		CreatedAt: issueQuery.Repository.Issue.CreatedAt.Format(time.RFC3339),
		UpdatedAt: issueQuery.Repository.Issue.UpdatedAt.Format(time.RFC3339),
		Author:    author,
		Labels:    []byte("[]"),
		Body: sql.NullString{
			String: string(issueQuery.Repository.Issue.Body),
			Valid:  true,
		},
		Locked:    locked,
		Reactions: []byte("{}"),
	}

	if issueQuery.Repository.Issue.StateReason != nil {
		issue.StateReason = sql.NullString{
			String: string(*issueQuery.Repository.Issue.StateReason),
			Valid:  true,
		}
	}

	if issueQuery.Repository.Issue.ClosedAt != nil {
		issue.ClosedAt = sql.NullString{
			String: issueQuery.Repository.Issue.ClosedAt.Format(time.RFC3339),
			Valid:  true,
		}
	}

	var labels []string
	for _, edge := range issueQuery.Repository.Issue.Labels.Edges {
		labels = append(labels, string(edge.Node.Name))
	}

	if len(labels) > 0 {
		issue.Labels, err = json.Marshal(labels)
		if err != nil {
			return db.InsertIssueParams{}, nil, fmt.Errorf("failed to marshal labels: %w", err)
		}
	}

	reactions := map[string]int64{}
	for _, group := range issueQuery.Repository.Issue.ReactionGroups {
		count := int64(group.Reactors.TotalCount)
		if count == 0 {
			continue
		}
		reactions[string(group.Content)] = count
	}
	if len(reactions) > 0 {
		issue.Reactions, err = json.Marshal(reactions)
		if err != nil {
			return db.InsertIssueParams{}, nil, fmt.Errorf("failed to marshal reactions: %w", err)
		}
	}

	var issueCommentsQuery struct {
		Repository struct {
			Issue struct {
				Comments struct {
					PageInfo struct {
						HasNextPage githubv4.Boolean
						EndCursor   githubv4.String
					}
					Nodes []struct {
						ID        string
						CreatedAt githubv4.DateTime
						UpdatedAt githubv4.DateTime
						Author    *struct {
							Typename string `graphql:"__typename"`
							Login    githubv4.String
						}
						Body githubv4.String
					}
				} `graphql:"comments(first: 100, after: $cursor)"`
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	var comments []db.InsertIssueCommentParams

	variables = map[string]any{
		"owner":  githubv4.String(org),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(number),
		"cursor": (*githubv4.String)(nil),
	}

	for {
		err := gqlClient.Query(ctx, &issueCommentsQuery, variables)
		if err != nil {
			return db.InsertIssueParams{}, nil, fmt.Errorf("failed to query issue comments: %w", err)
		}

		for _, node := range issueCommentsQuery.Repository.Issue.Comments.Nodes {
			commentAuthor := ""
			if node.Author != nil {
				commentAuthor = string(node.Author.Login)
				if node.Author.Typename == "Bot" {
					commentAuthor = commentAuthor + "[bot]"
				}
			}

			comments = append(comments, db.InsertIssueCommentParams{
				IssueNumber: number,
				ID:          node.ID,
				CreatedAt:   node.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   node.UpdatedAt.Format(time.RFC3339),
				Author:      commentAuthor,
				Body: sql.NullString{
					String: string(node.Body),
					Valid:  true,
				},
			})
		}

		if !issueCommentsQuery.Repository.Issue.Comments.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(issueCommentsQuery.Repository.Issue.Comments.PageInfo.EndCursor)
	}

	return issue, comments, nil
}
