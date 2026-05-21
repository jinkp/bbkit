package mcp

import (
	"context"
	"log"
	"os"

	"github.com/jinkp/bbkit/internal/version"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// StartServer creates and starts the MCP stdio server.
// CRITICAL: This function MUST NOT write anything to stdout before calling server.ServeStdio.
// The MCP stdio transport owns stdout entirely.
func StartServer() error {
	// Redirect the default logger to stderr so no accidental stdout writes happen.
	log.SetOutput(os.Stderr)

	s := server.NewMCPServer(
		"bbkit",
		version.Version,
	)

	// Register all 18 tools
	registerTools(s)

	ctx := context.Background()
	_ = ctx

	return server.ServeStdio(s)
}

// registerTools registers all 18 bbkit MCP tools on the server.
func registerTools(s *server.MCPServer) {
	// Workspace-scoped tools
	s.AddTool(mcpgo.NewTool("bb_list_repos",
		mcpgo.WithDescription("List Bitbucket repositories in a workspace"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
	), handleListRepos)

	// PR tools
	s.AddTool(mcpgo.NewTool("bb_list_prs",
		mcpgo.WithDescription("List pull requests for a repository"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithString("state", mcpgo.Description("PR state filter: OPEN, MERGED, DECLINED (default: OPEN)")),
	), handleListPRs)

	s.AddTool(mcpgo.NewTool("bb_get_pr",
		mcpgo.WithDescription("Get details of a specific pull request"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handleGetPR)

	s.AddTool(mcpgo.NewTool("bb_pr_comments",
		mcpgo.WithDescription("List comments on a pull request"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handlePRComments)

	s.AddTool(mcpgo.NewTool("bb_pr_commits",
		mcpgo.WithDescription("List commits in a pull request"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handlePRCommits)

	s.AddTool(mcpgo.NewTool("bb_pr_files",
		mcpgo.WithDescription("List files changed in a pull request (diffstat)"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handlePRFiles)

	s.AddTool(mcpgo.NewTool("bb_pr_diff",
		mcpgo.WithDescription("Get the diff for a pull request"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handlePRDiff)

	s.AddTool(mcpgo.NewTool("bb_pr_checks",
		mcpgo.WithDescription("Get the build/pipeline status checks for a pull request"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handlePRChecks)

	s.AddTool(mcpgo.NewTool("bb_pr_reviewers",
		mcpgo.WithDescription("List reviewers of a pull request"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handlePRReviewers)

	// PR write tools
	s.AddTool(mcpgo.NewTool("bb_create_pr",
		mcpgo.WithDescription("Create a pull request (WRITE operation — modifies Bitbucket)"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithString("title", mcpgo.Description("Pull request title"), mcpgo.Required()),
		mcpgo.WithString("source", mcpgo.Description("Source branch name"), mcpgo.Required()),
		mcpgo.WithString("destination", mcpgo.Description("Destination branch name"), mcpgo.Required()),
		mcpgo.WithString("description", mcpgo.Description("Pull request description")),
		mcpgo.WithString("reviewers", mcpgo.Description("Comma-separated reviewer UUIDs or usernames")),
		mcpgo.WithBoolean("close_source_branch", mcpgo.Description("Close source branch after merge")),
	), handleCreatePR)

	s.AddTool(mcpgo.NewTool("bb_comment_pr",
		mcpgo.WithDescription("Add a comment to a pull request (WRITE operation — modifies Bitbucket)"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
		mcpgo.WithString("message", mcpgo.Description("Comment text to post"), mcpgo.Required()),
	), handleCommentPR)

	s.AddTool(mcpgo.NewTool("bb_update_pr",
		mcpgo.WithDescription("Update a pull request's title, description, destination, or reviewers (WRITE operation — modifies Bitbucket). At least one optional field must be provided."),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
		mcpgo.WithString("title", mcpgo.Description("New pull request title")),
		mcpgo.WithString("description", mcpgo.Description("New pull request description")),
		mcpgo.WithString("destination", mcpgo.Description("New destination branch name")),
		mcpgo.WithString("reviewers", mcpgo.Description("Comma-separated reviewer UUIDs or usernames (replaces existing reviewers)")),
	), handleUpdatePR)

	s.AddTool(mcpgo.NewTool("bb_approve_pr",
		mcpgo.WithDescription("Approve a pull request (WRITE operation — modifies Bitbucket). Idempotent: re-approving an already-approved PR is safe."),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
	), handleApprovePR)

	s.AddTool(mcpgo.NewTool("bb_create_pr_task",
		mcpgo.WithDescription("Create a task on a pull request (WRITE operation — modifies Bitbucket)"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
		mcpgo.WithString("message", mcpgo.Description("Task description text"), mcpgo.Required()),
	), handleCreatePRTask)

	s.AddTool(mcpgo.NewTool("bb_resolve_pr_task",
		mcpgo.WithDescription("Resolve a task on a pull request (WRITE operation — modifies Bitbucket). task_id can be obtained from bb_get_pr or a previous bb_pr_tasks call (if available)."),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("pr_id", mcpgo.Description("Pull request ID"), mcpgo.Required()),
		mcpgo.WithInteger("task_id", mcpgo.Description("Task ID to resolve"), mcpgo.Required()),
	), handleResolvePRTask)

	// Branch tools
	s.AddTool(mcpgo.NewTool("bb_list_branches",
		mcpgo.WithDescription("List branches in a repository"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
	), handleListBranches)

	s.AddTool(mcpgo.NewTool("bb_stale_branches",
		mcpgo.WithDescription("List branches with no commits in the last N days"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
		mcpgo.WithInteger("days", mcpgo.Description("Number of days to consider stale (default: 30)")),
	), handleStaleBranches)

	// Pipeline tool
	s.AddTool(mcpgo.NewTool("bb_list_pipelines",
		mcpgo.WithDescription("List recent pipelines for a repository"),
		mcpgo.WithString("workspace", mcpgo.Description("Bitbucket workspace slug (overrides env/config)")),
		mcpgo.WithString("repo", mcpgo.Description("Repository slug (overrides env/config)")),
	), handleListPipelines)
}
