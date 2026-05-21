package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/git"
	"github.com/jinkp/bbkit/internal/secrets"
	"github.com/jinkp/bbkit/internal/services"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

// resolveWorkspace resolves workspace from: tool param → BITBUCKET_WORKSPACE env → config file → git remote.
func resolveWorkspace(req mcpgo.CallToolRequest) (string, error) {
	ws := req.GetString("workspace", "")
	if ws == "" {
		ws = os.Getenv("BITBUCKET_WORKSPACE")
	}
	if ws == "" {
		cfg, err := config.Load()
		if err == nil && cfg != nil {
			ws = cfg.Workspace
		}
	}
	if ws == "" {
		remote, _ := git.InferFromGitRemote()
		if remote != nil {
			ws = remote.Workspace
		}
	}
	if ws == "" {
		return "", fmt.Errorf("no workspace configured — pass workspace param, set BITBUCKET_WORKSPACE env var, or run from a Bitbucket git repo")
	}
	return strings.TrimSpace(ws), nil
}

// resolveWorkspaceRepo resolves both workspace and repo from: tool params → env → config → git remote.
// repo may be empty for workspace-scoped tools.
func resolveWorkspaceRepo(req mcpgo.CallToolRequest) (workspace, repo string, err error) {
	workspace, err = resolveWorkspace(req)
	if err != nil {
		return "", "", err
	}

	repo = req.GetString("repo", "")
	if repo == "" {
		repo = os.Getenv("BITBUCKET_REPO")
	}
	if repo == "" {
		cfg, cfgErr := config.Load()
		if cfgErr == nil && cfg != nil {
			repo = cfg.Repo
		}
	}
	if repo == "" {
		remote, _ := git.InferFromGitRemote()
		if remote != nil {
			repo = remote.RepoSlug
		}
	}
	if repo == "" {
		return "", "", fmt.Errorf("no repository configured — pass repo param, set BITBUCKET_REPO env var, or run from a Bitbucket git repo")
	}

	return strings.TrimSpace(workspace), strings.TrimSpace(repo), nil
}

// createClient creates a Bitbucket client using stored config and secrets.
func createClient() (*bitbucket.Client, error) {
	creds, err := secrets.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}
	if creds == nil || creds.Username == "" || creds.APIToken == "" {
		return nil, fmt.Errorf("not authenticated — run `bbk auth login` or set BITBUCKET_USERNAME and BITBUCKET_API_TOKEN")
	}
	return bitbucket.NewClient(creds.Username, creds.APIToken), nil
}

// toJSON marshals v to a JSON string, returning an error result on failure.
func toJSON(v any) (*mcpgo.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcpgo.NewToolResultText(string(data)), nil
}

// toolErr returns an MCP error result (isError=true) with the given message.
func toolErr(msg string) *mcpgo.CallToolResult {
	return mcpgo.NewToolResultError(msg)
}

// --- Tool handlers ---

func handleListRepos(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, err := resolveWorkspace(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	repos, err := services.ListRepositories(ctx, client, ws)
	if err != nil {
		return toolErr(fmt.Sprintf("list repositories: %v", err)), nil
	}
	return toJSON(repos)
}

func handleListPRs(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	state := req.GetString("state", "")
	opts := services.PrListOptions{}
	if state != "" {
		opts.States = []string{state}
	}

	prs, err := services.List(ctx, client, ws, repo, opts)
	if err != nil {
		return toolErr(fmt.Sprintf("list pull requests: %v", err)), nil
	}
	return toJSON(prs)
}

func handleGetPR(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	pr, err := services.Get(ctx, client, ws, repo, prID)
	if err != nil {
		return toolErr(fmt.Sprintf("get pull request: %v", err)), nil
	}
	return toJSON(pr)
}

func handlePRComments(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	comments, err := services.Comments(ctx, client, ws, repo, prID)
	if err != nil {
		return toolErr(fmt.Sprintf("list PR comments: %v", err)), nil
	}
	return toJSON(comments)
}

func handlePRCommits(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	commits, err := services.Commits(ctx, client, ws, repo, prID)
	if err != nil {
		return toolErr(fmt.Sprintf("list PR commits: %v", err)), nil
	}
	return toJSON(commits)
}

func handlePRFiles(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	files, err := services.Files(ctx, client, ws, repo, prID)
	if err != nil {
		return toolErr(fmt.Sprintf("list PR files: %v", err)), nil
	}
	return toJSON(files)
}

func handlePRDiff(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	diff, err := services.Diff(ctx, client, ws, repo, prID)
	if err != nil {
		return toolErr(fmt.Sprintf("get PR diff: %v", err)), nil
	}
	return mcpgo.NewToolResultText(diff), nil
}

func handlePRChecks(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	checks, err := services.Checks(ctx, client, ws, repo, prID)
	if err != nil {
		return toolErr(fmt.Sprintf("get PR checks: %v", err)), nil
	}
	return toJSON(checks)
}

func handlePRReviewers(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	reviewers, err := services.Reviewers(ctx, client, ws, repo, prID)
	if err != nil {
		return toolErr(fmt.Sprintf("get PR reviewers: %v", err)), nil
	}
	return toJSON(reviewers)
}

func handleListBranches(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	branches, err := services.ListBranches(ctx, client, ws, repo)
	if err != nil {
		return toolErr(fmt.Sprintf("list branches: %v", err)), nil
	}
	return toJSON(branches)
}

func handleStaleBranches(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	days := req.GetInt("days", 30)
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	branches, err := services.StaleBranches(ctx, client, ws, repo, days)
	if err != nil {
		return toolErr(fmt.Sprintf("list stale branches: %v", err)), nil
	}
	return toJSON(branches)
}

func handleListPipelines(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}
	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	pipelines, err := services.ListPipelines(ctx, client, ws, repo)
	if err != nil {
		return toolErr(fmt.Sprintf("list pipelines: %v", err)), nil
	}
	return toJSON(pipelines)
}

// --- PR write tool helpers ---

// splitReviewers splits a comma-separated reviewer string into a slice of trimmed,
// non-empty strings. Returns nil when the input is empty or whitespace-only.
func splitReviewers(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// --- PR write tool handlers ---

func handleCreatePR(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}

	title := req.GetString("title", "")
	source := req.GetString("source", "")
	destination := req.GetString("destination", "")
	if title == "" {
		return toolErr("title is required"), nil
	}
	if source == "" {
		return toolErr("source is required"), nil
	}
	if destination == "" {
		return toolErr("destination is required"), nil
	}

	description := req.GetString("description", "")
	reviewersRaw := req.GetString("reviewers", "")
	closeSourceBranch := req.GetBool("close_source_branch", false)

	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	payload := services.PrCreatePayload{
		Title:             title,
		Description:       description,
		SourceBranch:      source,
		DestinationBranch: destination,
		Reviewers:         splitReviewers(reviewersRaw),
		CloseSourceBranch: closeSourceBranch,
	}

	pr, err := services.Create(ctx, client, ws, repo, payload)
	if err != nil {
		return toolErr(fmt.Sprintf("create pull request: %v", err)), nil
	}
	return toJSON(pr)
}

func handleCommentPR(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	message := req.GetString("message", "")
	if message == "" {
		return toolErr("message is required"), nil
	}

	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	if err := services.AddComment(ctx, client, ws, repo, prID, message); err != nil {
		return toolErr(fmt.Sprintf("comment on pull request: %v", err)), nil
	}
	return mcpgo.NewToolResultText(fmt.Sprintf("Comment posted on PR #%d", prID)), nil
}

func handleUpdatePR(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}

	title := req.GetString("title", "")
	description := req.GetString("description", "")
	destination := req.GetString("destination", "")
	reviewersRaw := req.GetString("reviewers", "")

	if title == "" && description == "" && destination == "" && reviewersRaw == "" {
		return toolErr("bb_update_pr requires at least one of: title, description, destination, reviewers"), nil
	}

	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	payload := services.PrUpdatePayload{
		Title:             title,
		Description:       description,
		DestinationBranch: destination,
		Reviewers:         splitReviewers(reviewersRaw), // nil when reviewersRaw is empty
	}

	pr, err := services.Update(ctx, client, ws, repo, prID, payload)
	if err != nil {
		return toolErr(fmt.Sprintf("update pull request: %v", err)), nil
	}
	return toJSON(pr)
}

func handleApprovePR(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}

	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	if err := services.Approve(ctx, client, ws, repo, prID); err != nil {
		return toolErr(fmt.Sprintf("approve pull request: %v", err)), nil
	}
	return mcpgo.NewToolResultText(fmt.Sprintf("PR #%d approved", prID)), nil
}

func handleCreatePRTask(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	message := req.GetString("message", "")
	if message == "" {
		return toolErr("message is required"), nil
	}

	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	task, err := services.CreateTask(ctx, client, ws, repo, prID, message)
	if err != nil {
		return toolErr(fmt.Sprintf("create PR task: %v", err)), nil
	}
	return toJSON(task)
}

func handleResolvePRTask(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	ws, repo, err := resolveWorkspaceRepo(req)
	if err != nil {
		return toolErr(err.Error()), nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID <= 0 {
		return toolErr("pr_id must be a positive integer"), nil
	}
	taskID := req.GetInt("task_id", 0)
	if taskID <= 0 {
		return toolErr("task_id must be a positive integer"), nil
	}

	client, err := createClient()
	if err != nil {
		return toolErr(err.Error()), nil
	}

	if err := services.ResolveTask(ctx, client, ws, repo, prID, taskID); err != nil {
		return toolErr(fmt.Sprintf("resolve PR task: %v", err)), nil
	}
	return mcpgo.NewToolResultText(fmt.Sprintf("Task #%d on PR #%d resolved", taskID, prID)), nil
}
