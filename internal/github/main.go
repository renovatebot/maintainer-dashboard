package github

import (
	"context"
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v81/github"
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

/*
- List all Discussions (number)
{
  repository(owner: "renovatebot", name: "renovate") {
    discussions(first:50, orderBy:{field:UPDATED_AT, direction:DESC}){
      pageInfo {
        hasNextPage
      }
      edges {
        node {
          number
        }
      }
    }
  }
}
- What don't we have?
  - Sync those fully

- What do we have?
  - what's newer than what we have?
*/

type ListDiscussionNumbersResult struct {
	Number int64
	Error  error
}

func ListDiscussionNumbers(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, org, repo string, results chan<- ListDiscussionNumbersResult, done chan<- struct{}) {
	defer close(done)

	var q struct {
		Repository struct {
			Discussions struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
				Edges []struct {
					Node struct {
						Number int64
					}
				}
			} `graphql:"discussions(first: 100, orderBy: {field: CREATED_AT, direction: ASC}, after: $cursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]any{
		"owner":  githubv4.String(org),
		"name":   githubv4.String(repo),
		"cursor": (*githubv4.String)(nil), // Null after argument to get first page.
	}

	// // Get comments from all pages.
	// var allComments []comment
	for {
		err := gqlClient.Query(ctx, &q, variables)
		if err != nil {
			results <- ListDiscussionNumbersResult{
				Error: err,
			}
			// TODO
			// close(done)
			return
			// return err
		}
		for _, edge := range q.Repository.Discussions.Edges {
			results <- ListDiscussionNumbersResult{
				Number: edge.Node.Number,
			}
		}

		if !q.Repository.Discussions.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(q.Repository.Discussions.PageInfo.EndCursor)
	}
}
