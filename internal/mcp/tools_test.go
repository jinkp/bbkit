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

// --- splitReviewers tests ---

func TestSplitReviewers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace-only returns nil",
			input: "   ",
			want:  nil,
		},
		{
			name:  "only commas and spaces returns nil",
			input: " , , ",
			want:  nil,
		},
		{
			name:  "single reviewer",
			input: "alice",
			want:  []string{"alice"},
		},
		{
			name:  "single reviewer with surrounding whitespace",
			input: "  alice  ",
			want:  []string{"alice"},
		},
		{
			name:  "multiple reviewers",
			input: "alice,bob,carol",
			want:  []string{"alice", "bob", "carol"},
		},
		{
			name:  "multiple reviewers with mixed whitespace",
			input: " alice , bob , carol ",
			want:  []string{"alice", "bob", "carol"},
		},
		{
			name:  "UUID format with braces",
			input: "{uuid1},{uuid2}",
			want:  []string{"{uuid1}", "{uuid2}"},
		},
		{
			name:  "UUID format with surrounding whitespace",
			input: " {uuid1} , {uuid2} ",
			want:  []string{"{uuid1}", "{uuid2}"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitReviewers(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

// --- handleCreatePR tests ---

func TestHandleCreatePRReturnsErrorForMissingWorkspace(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "")
	t.Setenv("BITBUCKET_REPO", "")

	tmpDir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(tmpDir))

	req := makeRequest(map[string]any{
		"title":       "My PR",
		"source":      "feature",
		"destination": "main",
	})
	result, err := handleCreatePR(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError, "expected error result when workspace is missing")
}

func TestHandleCreatePRReturnsErrorForEmptyTitle(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	tests := []struct {
		name        string
		args        map[string]any
		errContains string
	}{
		{
			name:        "missing title",
			args:        map[string]any{"source": "feature", "destination": "main"},
			errContains: "title is required",
		},
		{
			name:        "missing source",
			args:        map[string]any{"title": "My PR", "destination": "main"},
			errContains: "source is required",
		},
		{
			name:        "missing destination",
			args:        map[string]any{"title": "My PR", "source": "feature"},
			errContains: "destination is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := handleCreatePR(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.IsError, "expected error result")
			text := result.Content[0].(mcpgo.TextContent).Text
			require.Contains(t, text, tt.errContains)
		})
	}
}

// --- handleCommentPR tests ---

func TestHandleCommentPRReturnsErrorForInvalidPRID(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	tests := []struct {
		name        string
		args        map[string]any
		errContains string
	}{
		{
			name:        "pr_id negative",
			args:        map[string]any{"pr_id": -1, "message": "hello"},
			errContains: "pr_id must be a positive integer",
		},
		{
			name:        "pr_id zero",
			args:        map[string]any{"pr_id": 0, "message": "hello"},
			errContains: "pr_id must be a positive integer",
		},
		{
			name:        "empty message",
			args:        map[string]any{"pr_id": 5, "message": ""},
			errContains: "message is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := handleCommentPR(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.IsError, "expected error result")
			text := result.Content[0].(mcpgo.TextContent).Text
			require.Contains(t, text, tt.errContains)
		})
	}
}

// --- handleUpdatePR tests ---

func TestHandleUpdatePRReturnsErrorForNoPRID(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	req := makeRequest(map[string]any{"pr_id": 0, "title": "New Title"})
	result, err := handleUpdatePR(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError, "expected error result for pr_id=0")

	text := result.Content[0].(mcpgo.TextContent).Text
	require.Contains(t, text, "pr_id must be a positive integer")
}

func TestHandleUpdatePRReturnsErrorForNoFields(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	// pr_id is valid but no update fields provided
	req := makeRequest(map[string]any{"pr_id": 5})
	result, err := handleUpdatePR(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError, "expected error when no update fields provided")

	text := result.Content[0].(mcpgo.TextContent).Text
	require.Contains(t, text, "at least one of")
}

// --- handleApprovePR tests ---

func TestHandleApprovePRReturnsErrorForInvalidPRID(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	tests := []struct {
		name  string
		prID  int
	}{
		{name: "negative pr_id", prID: -1},
		{name: "zero pr_id", prID: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(map[string]any{"pr_id": tt.prID})
			result, err := handleApprovePR(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.IsError, "expected error result for pr_id=%d", tt.prID)
			text := result.Content[0].(mcpgo.TextContent).Text
			require.Contains(t, text, "pr_id must be a positive integer")
		})
	}
}

// --- handleCreatePRTask tests ---

func TestHandleCreatePRTaskReturnsErrorForMissingWorkspace(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "")
	t.Setenv("BITBUCKET_REPO", "")

	tmpDir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(tmpDir))

	req := makeRequest(map[string]any{"pr_id": 5, "message": "Do the thing"})
	result, err := handleCreatePRTask(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError, "expected error result when workspace is missing")
}

func TestHandleCreatePRTaskReturnsErrorForInvalidArgs(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	tests := []struct {
		name        string
		args        map[string]any
		errContains string
	}{
		{
			name:        "pr_id zero",
			args:        map[string]any{"pr_id": 0, "message": "task"},
			errContains: "pr_id must be a positive integer",
		},
		{
			name:        "empty message",
			args:        map[string]any{"pr_id": 5, "message": ""},
			errContains: "message is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := handleCreatePRTask(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.IsError, "expected error result")
			text := result.Content[0].(mcpgo.TextContent).Text
			require.Contains(t, text, tt.errContains)
		})
	}
}

// --- handleResolvePRTask tests ---

func TestHandleResolvePRTaskReturnsErrorForInvalidTaskID(t *testing.T) {
	setEnvCredentials(t)
	t.Setenv("BITBUCKET_WORKSPACE", "myworkspace")
	t.Setenv("BITBUCKET_REPO", "myrepo")

	tests := []struct {
		name        string
		args        map[string]any
		errContains string
	}{
		{
			name:        "task_id zero",
			args:        map[string]any{"pr_id": 5, "task_id": 0},
			errContains: "task_id must be a positive integer",
		},
		{
			name:        "task_id negative",
			args:        map[string]any{"pr_id": 5, "task_id": -1},
			errContains: "task_id must be a positive integer",
		},
		{
			name:        "pr_id zero",
			args:        map[string]any{"pr_id": 0, "task_id": 5},
			errContains: "pr_id must be a positive integer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := handleResolvePRTask(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.IsError, "expected error result")
			text := result.Content[0].(mcpgo.TextContent).Text
			require.Contains(t, text, tt.errContains)
		})
	}
}
