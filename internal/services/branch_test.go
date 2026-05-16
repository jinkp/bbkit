package services

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListBranchesReturnsBranches(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/2.0/repositories/workspace/repo/refs/branches", req.URL.Path)
		return jsonResponse(http.StatusOK, `{"values":[{"name":"main","target":{"hash":"abcdef1234567890","date":"2026-04-28T00:00:00Z","author":{"user":{"display_name":"Jane Doe"}}}}]}`), nil
	})

	branches, err := ListBranches(context.Background(), client, "workspace", "repo")
	require.NoError(t, err)
	require.Len(t, branches, 1)
	require.Equal(t, "main", branches[0].Name)
}

func TestStaleBranchesFiltersOlderThanNDays(t *testing.T) {
	oldDate := time.Now().Add(-90 * 24 * time.Hour).UTC().Format(time.RFC3339)
	newDate := time.Now().Add(-10 * 24 * time.Hour).UTC().Format(time.RFC3339)
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"values":[{"name":"stale","target":{"hash":"abc","date":"`+oldDate+`","author":{"user":{"display_name":"Jane Doe"}}}},{"name":"fresh","target":{"hash":"def","date":"`+newDate+`","author":{"user":{"display_name":"John Doe"}}}}]}`), nil
	})

	branches, err := StaleBranches(context.Background(), client, "workspace", "repo", 60)
	require.NoError(t, err)
	require.Len(t, branches, 1)
	require.Equal(t, "stale", branches[0].Name)
}

func TestStaleBranchesDefaultsTo30Days(t *testing.T) {
	oldDate := time.Now().Add(-31 * 24 * time.Hour).UTC().Format(time.RFC3339)
	newDate := time.Now().Add(-5 * 24 * time.Hour).UTC().Format(time.RFC3339)
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"values":[{"name":"stale","target":{"hash":"abc","date":"`+oldDate+`","author":{"user":{"display_name":"Jane Doe"}}}},{"name":"fresh","target":{"hash":"def","date":"`+newDate+`","author":{"user":{"display_name":"John Doe"}}}}]}`), nil
	})

	branches, err := StaleBranches(context.Background(), client, "workspace", "repo", 0)
	require.NoError(t, err)
	require.Len(t, branches, 1)
	require.Equal(t, "stale", branches[0].Name)
}
