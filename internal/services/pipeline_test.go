package services

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListPipelinesReturnsPipelines(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/2.0/repositories/workspace/repo/pipelines/", req.URL.Path)
		require.Equal(t, "-created_on", req.URL.Query().Get("sort"))
		require.Equal(t, "20", req.URL.Query().Get("pagelen"))
		return jsonResponse(http.StatusOK, `{"values":[{"uuid":"{123}","build_number":42,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref_name":"main","ref_type":"branch"},"created_on":"2026-04-28T00:00:00Z","links":{"self":{"href":"https://api.bitbucket.org/2.0/pipelines/123"}}}]}`), nil
	})

	pipelines, err := ListPipelines(context.Background(), client, "workspace", "repo")
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	require.Equal(t, 42, pipelines[0].BuildNumber)
}

func TestListPipelinesReturnsAPIError(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusForbidden, `{"type":"error","error":{"message":"Repository has no Pipelines configuration"}}`), nil
	})

	_, err := ListPipelines(context.Background(), client, "workspace", "repo")
	require.Error(t, err)
}

func TestRunPipelineReturnsAPIError(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"type":"error","error":{"message":"Branch 'nonexistent' not found"}}`), nil
	})

	_, err := RunPipeline(context.Background(), client, "workspace", "repo", "nonexistent")
	require.Error(t, err)
}

func TestRunPipelineTriggersCorrectBranch(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/2.0/repositories/workspace/repo/pipelines/", req.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&payload))
		target, ok := payload["target"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "develop", target["ref_name"])
		return jsonResponse(http.StatusOK, `{"uuid":"{123}","build_number":42,"state":{"name":"PENDING"},"target":{"ref_name":"develop","ref_type":"branch"},"created_on":"2026-04-28T00:00:00Z","links":{"self":{"href":"https://api.bitbucket.org/2.0/pipelines/123"}}}`), nil
	})

	pipeline, err := RunPipeline(context.Background(), client, "workspace", "repo", "develop")
	require.NoError(t, err)
	require.Equal(t, "develop", pipeline.Target.RefName)
}
