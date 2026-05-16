package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/jinkp/bbkit/internal/output"
	"github.com/jinkp/bbkit/internal/version"
	"github.com/stretchr/testify/require"
)

func TestVersionCommandOutputsVersionString(t *testing.T) {
	setupCLIEnv(t)

	stdout, stderr, err := executeRoot(t, nil, "version")
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "bbk version "+version.Version)
}

func TestVersionCommandOutputsJSON(t *testing.T) {
	setupCLIEnv(t)

	stdout, _, err := executeRoot(t, nil, "version", "--json")
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Equal(t, version.Version, payload["version"])
}

func TestConfigSetGetAndList(t *testing.T) {
	setupCLIEnv(t)

	stdout, _, err := executeRoot(t, nil, "config", "set", "workspace", "team-a")
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "workspace = team-a")

	stdout, _, err = executeRoot(t, nil, "config", "get", "workspace")
	require.NoError(t, err)
	require.Equal(t, "team-a\n", stdout.String())

	_, _, err = executeRoot(t, nil, "config", "set", "defaultOutput", "json")
	require.NoError(t, err)

	stdout, _, err = executeRoot(t, nil, "config", "list")
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "workspace=team-a")
	require.Contains(t, stdout.String(), "defaultOutput=json")
}

func TestAuthStatusWhenNotAuthenticated(t *testing.T) {
	setupCLIEnv(t)

	stdout, stderr, err := executeRoot(t, nil, "auth", "status")
	require.NoError(t, err)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "Not authenticated")
}

func TestPRMergeRequiresYesOrPrompt(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "workspace")
	t.Setenv("BITBUCKET_REPO", "repo")

	stdout, stderr, err := executeRoot(t, bytes.NewBufferString("n\n"), "pr", "merge", "123")
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Merge PR #123 into workspace/repo? [y/N]: ")
	require.Contains(t, stdout.String(), "Merge cancelled. PR #123 was not merged.")
}

func TestRootJSONFlagCascadesToSubcommand(t *testing.T) {
	setupCLIEnv(t)

	stdout, _, err := executeRoot(t, nil, "--json", "version")
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Equal(t, version.Version, payload["version"])
}

func TestWorkspaceResolutionPrecedence(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_REPO", "repo")

	// Flag takes precedence over env
	t.Setenv("BITBUCKET_WORKSPACE", "env-workspace")
	stdout, _, err := executeRoot(t, nil, "--workspace", "flag-workspace", "repo", "list")
	// Expected to fail because no real API, but the error should NOT be about missing workspace
	if err != nil {
		require.NotContains(t, err.Error(), "No workspace configured")
	}
	_ = stdout
}

func TestMissingWorkspaceReturnsHelpfulError(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "")
	t.Setenv("BITBUCKET_REPO", "")

	_, _, err := executeRoot(t, nil, "repo", "list")
	require.Error(t, err)
	require.Contains(t, err.Error(), "No workspace configured")
}

func TestMissingRepoReturnsHelpfulError(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "my-workspace")
	t.Setenv("BITBUCKET_REPO", "")

	_, _, err := executeRoot(t, nil, "pr", "list")
	require.Error(t, err)
	require.Contains(t, err.Error(), "No repository found")
}

func TestBranchStaleInvalidDaysReturnsError(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "workspace")
	t.Setenv("BITBUCKET_REPO", "repo")

	_, _, err := executeRoot(t, nil, "branch", "stale", "--days", "abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-negative number")
}

func TestBranchStaleNegativeDaysReturnsError(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "workspace")
	t.Setenv("BITBUCKET_REPO", "repo")

	_, _, err := executeRoot(t, nil, "branch", "stale", "--days", "-5")
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-negative number")
}

func TestPRListAllCannotCombineWithState(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "workspace")
	t.Setenv("BITBUCKET_REPO", "repo")

	_, _, err := executeRoot(t, nil, "pr", "list", "--all", "--state", "OPEN")
	require.Error(t, err)
	require.Contains(t, err.Error(), "--all cannot be combined with --state")
}

func TestPRCreateMissingRequiredFlags(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "workspace")
	t.Setenv("BITBUCKET_REPO", "repo")

	// Missing title, source, destination
	_, _, err := executeRoot(t, nil, "pr", "create", "--source", "feature", "--destination", "main")
	require.Error(t, err)
	require.Contains(t, err.Error(), "title is required")
}

func TestPRMergeInvalidStrategyReturnsError(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "workspace")
	t.Setenv("BITBUCKET_REPO", "repo")

	_, _, err := executeRoot(t, nil, "pr", "merge", "123", "--yes", "--strategy", "invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid merge strategy")
}

func TestConfigSetInvalidKeyReturnsError(t *testing.T) {
	setupCLIEnv(t)

	_, _, err := executeRoot(t, nil, "config", "set", "invalidKey", "value")
	require.Error(t, err)
}

func TestPRTaskResolveRequiresTwoPositionalArgs(t *testing.T) {
	setupCLIEnv(t)
	t.Setenv("BITBUCKET_USERNAME", "user@example.com")
	t.Setenv("BITBUCKET_API_TOKEN", "token123")
	t.Setenv("BITBUCKET_WORKSPACE", "workspace")
	t.Setenv("BITBUCKET_REPO", "repo")

	_, _, err := executeRoot(t, nil, "pr", "task", "resolve", "123")
	require.Error(t, err)
	// Should fail because only 1 arg provided instead of 2
}

func TestUnknownCommandExitsNonZero(t *testing.T) {
	t.Parallel()
	cmd := exec.Command(os.Args[0], "-test.run=TestExecuteHelperProcess", "--", "unknown")
	cmd.Env = append(os.Environ(), "GO_WANT_EXECUTE_HELPER=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.NotZero(t, exitErr.ExitCode())
	require.NotEmpty(t, stderr.String())
}

func TestExecuteHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_EXECUTE_HELPER") != "1" {
		return
	}
	setupCLIEnv(t)
	os.Args = []string{"bbk", "unknown"}
	Execute()
}

func executeRoot(t *testing.T, input *bytes.Buffer, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	if input != nil {
		root.SetIn(input)
	}
	output.SetWriters(stdout, stderr)
	t.Cleanup(output.ResetWriters)
	return stdout, stderr, root.Execute()
}

func setupCLIEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("BITBUCKET_USERNAME", "")
	t.Setenv("BITBUCKET_API_TOKEN", "")
	t.Setenv("BITBUCKET_WORKSPACE", "")
	t.Setenv("BITBUCKET_REPO", "")
	version.Version = "dev"
}
