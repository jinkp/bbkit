package services

import (
	"context"
	"fmt"

	"github.com/jinkp/bbkit/internal/bitbucket"
)

func ListRepositories(ctx context.Context, client *bitbucket.Client, workspace string) ([]bitbucket.Repository, error) {
	return bitbucket.Paginate[bitbucket.Repository](ctx, client, fmt.Sprintf("/repositories/%s", workspace))
}
