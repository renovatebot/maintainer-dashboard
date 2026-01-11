package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v81/github"
	"github.com/renovatebot/maintainer-dashboard/internal/db"
	"github.com/shurcooL/githubv4"
)

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
							Login githubv4.String
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
									Login githubv4.String
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
			comments = append(comments, db.InsertDiscussionCommentParams{
				DiscussionNumber: number,
				ID:               node.ID,
				CreatedAt:        node.CreatedAt.Format(time.RFC3339),
				UpdatedAt:        node.UpdatedAt.Format(time.RFC3339),
				Author:           string(node.Author.Login),
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
				comments = append(comments, db.InsertDiscussionCommentParams{
					DiscussionNumber: number,
					ID:               reply.ID,
					CreatedAt:        reply.CreatedAt.Format(time.RFC3339),
					UpdatedAt:        reply.UpdatedAt.Format(time.RFC3339),
					Author:           string(reply.Author.Login),
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
		return time.Time{}, fmt.Errorf("failed to **??**: %w", err)
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
		return nil, nil, fmt.Errorf("failed to **??**: %w", err)
	}
	for _, edge := range q.Repository.Discussions.Edges {
		results = append(results, MostRecentlyUpdatedDiscussion{
			Number:    edge.Node.Number,
			UpdatedAt: edge.Node.UpdatedAt.Time,
		})
	}

	return results, &q.Repository.Discussions.PageInfo.EndCursor, nil
}
