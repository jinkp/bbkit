package services

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jinkp/bbkit/internal/bitbucket"
)

func ListPipelines(ctx context.Context, client *bitbucket.Client, workspace, repo string) ([]bitbucket.Pipeline, error) {
	query := url.Values{}
	query.Set("sort", "-created_on")
	query.Set("pagelen", "20")

	return bitbucket.Paginate[bitbucket.Pipeline](ctx, client, fmt.Sprintf("/repositories/%s/%s/pipelines/?%s", workspace, repo, query.Encode()))
}

func RunPipeline(ctx context.Context, client *bitbucket.Client, workspace, repo, branch string) (*bitbucket.Pipeline, error) {
	var pipeline bitbucket.Pipeline
	err := client.Post(ctx, fmt.Sprintf("/repositories/%s/%s/pipelines/", workspace, repo), map[string]any{
		"target": map[string]any{
			"ref_type": "branch",
			"type":     "pipeline_ref_target",
			"ref_name": branch,
		},
	}, &pipeline)
	if err != nil {
		return nil, err
	}

	return &pipeline, nil
}
