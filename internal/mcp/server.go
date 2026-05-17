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

	// Register all 12 tools
	registerTools(s)

	ctx := context.Background()
	_ = ctx

	return server.ServeStdio(s)
}

// registerTools registers all 12 bbkit MCP tools on the server.
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
