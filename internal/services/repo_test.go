package services

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListRepositoriesReturnsWorkspaceRepos(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/2.0/repositories/workspace", req.URL.Path)
		return jsonResponse(http.StatusOK, `{"values":[{"slug":"repo-one","name":"Repo One","full_name":"workspace/repo-one","updated_on":"2026-04-28T00:00:00Z"}]}`), nil
	})

	repos, err := ListRepositories(context.Background(), client, "workspace")
	require.NoError(t, err)
	require.Len(t, repos, 1)
	require.Equal(t, "repo-one", repos[0].Slug)
}

func TestListRepositoriesPaginates(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.RawQuery == "page=2" {
			return jsonResponse(http.StatusOK, `{"values":[{"slug":"repo-two","name":"Repo Two","full_name":"workspace/repo-two","updated_on":"2026-04-29T00:00:00Z"}]}`), nil
		}
		return jsonResponse(http.StatusOK, `{"values":[{"slug":"repo-one","name":"Repo One","full_name":"workspace/repo-one","updated_on":"2026-04-28T00:00:00Z"}],"next":"https://api.bitbucket.org/2.0/repositories/workspace?page=2"}`), nil
	})

	repos, err := ListRepositories(context.Background(), client, "workspace")
	require.NoError(t, err)
	require.Len(t, repos, 2)
	require.Equal(t, 2, requests)
}
