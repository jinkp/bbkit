package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jinkp/bbkit/internal/bitbucket"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// --- HTTP mock helpers (mirrors internal/services/test_helpers_test.go) ---

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// newTestHTTPClient replaces http.DefaultTransport for the duration of the test
// and returns a Bitbucket client that will use the mock transport.
func newTestHTTPClient(t *testing.T, fn roundTripFunc) *bitbucket.Client {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() { http.DefaultTransport = original })
	return bitbucket.NewClient("user", "token")
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func bodyResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

// makeRequest builds a CallToolRequest with the given arguments map.
func makeRequest(args map[string]any) mcpgo.CallToolRequest {
	return mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Arguments: args,
		},
	}
}

// --- resolveWorkspace tests ---

func TestResolveWorkspaceParamWinsOverEnv(t *testing.T) {
	t.Setenv("BITBUCKET_WORKSPACE", "env-workspace")

	req := makeRequest(map[string]any{"workspace": "param-workspace"})
	ws, err := resolveWorkspace(req)
	require.NoError(t, err)
	require.Equal(t, "param-workspace", ws, "tool param should win over env var")
}

func TestResolveWorkspaceEnvWinsOverConfig(t *testing.T) {
	// Ensure param is empty so env is checked.
	t.Setenv("BITBUCKET_WORKSPACE", "env-workspace")
	// Config file reads from OS path — env takes priority before config is even read.

	req := makeRequest(map[string]any{})
	ws, err := resolveWorkspace(req)
	require.NoError(t, err)
	require.Equal(t, "env-workspace", ws, "env var should win over config")
}

func TestResolveWorkspaceReturnsErrorWhenMissing(t *testing.T) {
	// Clear env so no workspace can be resolved.
	t.Setenv("BITBUCKET_WORKSPACE", "")

	// Use a temp dir as cwd with no git remote to ensure git inference also fails.
	// On Windows: register the chdir-restore cleanup AFTER TempDir so it runs
	// FIRST (LIFO order), releasing the dir before TempDir tries to remove it.
	tmpDir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(tmpDir))

	req := makeRequest(map[string]any{})
	_, err = resolveWorkspace(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no workspace configured")
}

// --- resolveWorkspaceRepo tests ---

func TestResolveWorkspaceRepoReturnsErrorWhenWorkspaceMissing(t *testing.T) {
	t.Setenv("BITBUCKET_WORKSPACE", "")
	t.Setenv("BITBUCKET_REPO", "")

	tmpDir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(tmpDir))

	req := makeRequest(map[string]any{})
	_, _, err = resolveWorkspaceRepo(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no workspace configured")
}

// --- Tool handler tests with mocked HTTP ---

// setEnvCredentials injects env-based credentials so createClient() succeeds
// without a real keyring or credential file.
func setEnvCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("BITBUCKET_USERNAME", "testuser")
	t.Setenv("BITBUCKET_API_TOKEN", "testtoken")
}

func TestHandleListReposReturnsRepos(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")

	_ = newTestHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/2.0/repositories/myworkspace", req.URL.Path)
		return jsonResponse(http.StatusOK, `{"values":[{"slug":"repo-a","name":"Repo A","full_name":"myworkspace/repo-a","updated_on":"2026-01-01T00:00:00Z"}]}`), nil
	})

	// Call the handler directly using the request; workspace comes from env.
	req := makeRequest(map[string]any{})
	result, err := handleListRepos(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsError, "expected successful result, got error: %v", result.Content)

	// The result text should be JSON containing the repo slug
	require.NotEmpty(t, result.Content)
	text := result.Content[0].(mcpgo.TextContent).Text
	require.Contains(t, text, "repo-a")
}

func TestHandleListPRsReturnsErrorForMissingWorkspace(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "")
	t.Setenv("BITBUCKET_REPO", "")

	tmpDir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(tmpDir))

	req := makeRequest(map[string]any{})
	result, err := handleListPRs(context.Background(), req)
	require.NoError(t, err) // handler never returns protocol-level error
	require.NotNil(t, result)
	require.True(t, result.IsError, "expected error result when workspace is missing")
}

func TestHandleGetPRReturnsErrorForInvalidPRID(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	req := makeRequest(map[string]any{"pr_id": 0})
	result, err := handleGetPR(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError, "expected error result for invalid pr_id")

	text := result.Content[0].(mcpgo.TextContent).Text
	require.Contains(t, text, "pr_id must be a positive integer")
}

func TestHandleListBranchesReturnsBranches(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	_ = newTestHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/2.0/repositories/myworkspace/myrepo/refs/branches", req.URL.Path)
		return jsonResponse(http.StatusOK, `{"values":[{"name":"main","target":{"hash":"abc123","date":"2026-01-01T00:00:00Z","author":{"user":{"display_name":"Alice"}}}}]}`), nil
	})

	req := makeRequest(map[string]any{})
	result, err := handleListBranches(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsError, "expected successful result, got error: %v", result.Content)

	text := result.Content[0].(mcpgo.TextContent).Text
	require.Contains(t, text, "main")
}
