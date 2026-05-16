package services

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/stretchr/testify/require"
)

func TestPRListReturnsPullRequests(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/2.0/repositories/workspace/repo/pullrequests", req.URL.Path)
		require.Equal(t, "OPEN", req.URL.Query().Get("state"))
		return jsonResponse(http.StatusOK, `{"values":[{"id":101,"title":"Add feature","state":"OPEN","source":{"branch":{"name":"feature/test"}},"destination":{"branch":{"name":"main"}},"author":{"display_name":"Jane Doe"},"created_on":"2026-04-28T00:00:00Z","updated_on":"2026-04-28T01:00:00Z","links":{"html":{"href":"https://bitbucket.org/ws/repo/pull-requests/101"}}}]}`), nil
	})

	prs, err := List(context.Background(), client, "workspace", "repo", PrListOptions{States: []string{"OPEN"}})
	require.NoError(t, err)
	require.Len(t, prs, 1)
	require.Equal(t, 101, prs[0].ID)
}

func TestPRGetReturnsSinglePullRequest(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/2.0/repositories/workspace/repo/pullrequests/77", req.URL.Path)
		return jsonResponse(http.StatusOK, `{"id":77,"title":"Checkout this branch","state":"OPEN","source":{"branch":{"name":"feature/checkout"}},"destination":{"branch":{"name":"main"}},"author":{"display_name":"Jane Doe"},"created_on":"2026-04-28T00:00:00Z","updated_on":"2026-04-28T01:00:00Z","links":{"html":{"href":"https://bitbucket.org/ws/repo/pull-requests/77"}}}`), nil
	})

	pr, err := Get(context.Background(), client, "workspace", "repo", 77)
	require.NoError(t, err)
	require.Equal(t, "feature/checkout", pr.Source.Branch.Name)
}

func TestPRMergeSendsCorrectPayload(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/2.0/repositories/workspace/repo/pullrequests/123/merge", req.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&payload))
		require.Equal(t, "pullrequest_merge_parameters", payload["type"])
		require.Equal(t, "squash", payload["merge_strategy"])
		require.Equal(t, "Merge feature X", payload["message"])
		return jsonResponse(http.StatusOK, `{"id":123,"title":"Ready to merge","state":"MERGED","source":{"branch":{"name":"feature/merge"}},"destination":{"branch":{"name":"main"}},"author":{"display_name":"Jane Doe"},"created_on":"2026-05-03T00:00:00Z","updated_on":"2026-05-03T01:00:00Z","links":{"html":{"href":"https://bitbucket.org/ws/repo/pull-requests/123"}}}`), nil
	})

	pr, err := Merge(context.Background(), client, "workspace", "repo", 123, PrMergePayload{Strategy: "squash", Message: "Merge feature X"})
	require.NoError(t, err)
	require.Equal(t, bitbucket.PullRequestStateMerged, pr.State)
}

func TestPRApproveSendsCorrectEndpoint(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/2.0/repositories/workspace/repo/pullrequests/123/approve", req.URL.Path)
		return jsonResponse(http.StatusOK, `{}`), nil
	})

	require.NoError(t, Approve(context.Background(), client, "workspace", "repo", 123))
}

func TestPRDeclineSendsCorrectEndpoint(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/2.0/repositories/workspace/repo/pullrequests/123/decline", req.URL.Path)
		return jsonResponse(http.StatusOK, `{}`), nil
	})

	require.NoError(t, Decline(context.Background(), client, "workspace", "repo", 123))
}

func TestPRListAllCannotCombineWithState(t *testing.T) {
	_, err := List(context.Background(), nil, "workspace", "repo", PrListOptions{
		All:    true,
		States: []string{"OPEN"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "--all cannot be combined with --state")
}

func TestPRCreateMissingTitle(t *testing.T) {
	_, err := Create(context.Background(), nil, "workspace", "repo", PrCreatePayload{
		SourceBranch:      "feature",
		DestinationBranch: "main",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "title is required")
}

func TestPRCreateMissingSource(t *testing.T) {
	_, err := Create(context.Background(), nil, "workspace", "repo", PrCreatePayload{
		Title:             "My PR",
		DestinationBranch: "main",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Source branch is required")
}

func TestPRCreateMissingDestination(t *testing.T) {
	_, err := Create(context.Background(), nil, "workspace", "repo", PrCreatePayload{
		Title:        "My PR",
		SourceBranch: "feature",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Destination branch is required")
}

func TestPRMergeInvalidStrategy(t *testing.T) {
	_, err := Merge(context.Background(), nil, "workspace", "repo", 1, PrMergePayload{Strategy: "invalid"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid merge strategy")
}

func TestPRMergeValidatesAllSixStrategies(t *testing.T) {
	strategies := []string{"merge_commit", "squash", "fast_forward", "squash_fast_forward", "rebase_fast_forward", "rebase_merge"}
	for _, strategy := range strategies {
		normalized, err := normalizeMergeStrategy(strategy)
		require.NoError(t, err, "strategy %s should be valid", strategy)
		require.NotEmpty(t, normalized)
	}
}

func TestPRUpdateMissingAllFields(t *testing.T) {
	_, err := Update(context.Background(), nil, "workspace", "repo", 1, PrUpdatePayload{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "At least one field must be provided")
}

func TestPRListWithReviewerFilter(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		q := req.URL.Query().Get("q")
		require.Contains(t, q, "reviewers.uuid")
		return jsonResponse(http.StatusOK, `{"values":[]}`), nil
	})

	_, err := List(context.Background(), client, "workspace", "repo", PrListOptions{Reviewer: "{user-uuid}"})
	require.NoError(t, err)
}

func TestPRListWithSourceAndTargetFilter(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		q := req.URL.Query().Get("q")
		require.Contains(t, q, `source.branch.name = "feature/x"`)
		require.Contains(t, q, `destination.branch.name = "main"`)
		return jsonResponse(http.StatusOK, `{"values":[]}`), nil
	})

	_, err := List(context.Background(), client, "workspace", "repo", PrListOptions{Source: "feature/x", Target: "main"})
	require.NoError(t, err)
}

func TestValidatePRListLimitRejectsInvalid(t *testing.T) {
	require.Error(t, ValidatePRListLimit("abc"))
	require.Error(t, ValidatePRListLimit("0"))
	require.Error(t, ValidatePRListLimit("-1"))
	require.NoError(t, ValidatePRListLimit("50"))
	require.NoError(t, ValidatePRListLimit(""))
}
