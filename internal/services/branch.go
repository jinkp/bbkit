package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jinkp/bbkit/internal/bitbucket"
)

const day = 24 * time.Hour

func ListBranches(ctx context.Context, client *bitbucket.Client, workspace, repo string) ([]bitbucket.Branch, error) {
	return bitbucket.Paginate[bitbucket.Branch](ctx, client, fmt.Sprintf("/repositories/%s/%s/refs/branches", workspace, repo))
}

func StaleBranches(ctx context.Context, client *bitbucket.Client, workspace, repo string, days int) ([]bitbucket.Branch, error) {
	if days <= 0 {
		days = 30
	}

	branches, err := ListBranches(ctx, client, workspace, repo)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-time.Duration(days) * day)
	stale := make([]bitbucket.Branch, 0, len(branches))
	for _, branch := range branches {
		commitTime, err := time.Parse(time.RFC3339, branch.Target.Date)
		if err != nil {
			continue
		}
		if commitTime.Before(cutoff) {
			stale = append(stale, branch)
		}
	}

	return stale, nil
}
