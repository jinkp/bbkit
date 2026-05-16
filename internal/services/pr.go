package services

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jinkp/bbkit/internal/bitbucket"
)

type PrListOptions struct {
	States   []string
	All      bool
	Author   string
	Reviewer string
	Source   string
	Target   string
	Query    string
	Limit    string
}

type PrCreatePayload struct {
	Title             string
	Description       string
	SourceBranch      string
	DestinationBranch string
	Reviewers         []string
	CloseSourceBranch bool
}

type PrUpdatePayload struct {
	Title             string
	Description       string
	DescriptionFile   string
	DestinationBranch string
	Reviewers         []string
}

type PrMergePayload struct {
	Strategy string
	Message  string
}

type reviewerRef struct {
	UUID     string `json:"uuid,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}

type prMergeRequest struct {
	Type          string `json:"type"`
	MergeStrategy string `json:"merge_strategy,omitempty"`
	Message       string `json:"message,omitempty"`
}

var allPRStates = []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED"}

func List(ctx context.Context, client *bitbucket.Client, workspace, repo string, opts PrListOptions) ([]bitbucket.PullRequest, error) {
	if opts.All && len(opts.States) > 0 {
		return nil, &bitbucket.CLIError{Message: "--all cannot be combined with --state.", ExitCode: 1}
	}

	query := url.Values{}

	states := opts.States
	if opts.All {
		states = allPRStates
	}
	for _, s := range states {
		query.Add("state", strings.ToUpper(strings.TrimSpace(s)))
	}

	if limit := strings.TrimSpace(opts.Limit); limit != "" {
		query.Set("pagelen", limit)
	}

	if q := buildPRQuery(opts); q != "" {
		query.Set("q", q)
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repo)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	return bitbucket.Paginate[bitbucket.PullRequest](ctx, client, path)
}

func Get(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) (*bitbucket.PullRequest, error) {
	var pr bitbucket.PullRequest
	if err := client.Get(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", workspace, repo, id), &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func Create(ctx context.Context, client *bitbucket.Client, workspace, repo string, payload PrCreatePayload) (*bitbucket.PullRequest, error) {
	resolved, err := resolveCreatePayload(payload)
	if err != nil {
		return nil, err
	}

	var pr bitbucket.PullRequest
	if err := client.Post(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repo), resolved, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func Update(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int, payload PrUpdatePayload) (*bitbucket.PullRequest, error) {
	resolved, err := resolveUpdatePayload(payload)
	if err != nil {
		return nil, err
	}

	var pr bitbucket.PullRequest
	if err := client.Patch(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", workspace, repo, id), resolved, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func Approve(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) error {
	return client.Post(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve", workspace, repo, id), map[string]any{}, nil)
}

func Decline(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) error {
	return client.Post(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline", workspace, repo, id), map[string]any{}, nil)
}

func Merge(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int, payload PrMergePayload) (*bitbucket.PullRequest, error) {
	strategy, err := normalizeMergeStrategy(payload.Strategy)
	if err != nil {
		return nil, err
	}

	message := strings.TrimSpace(payload.Message)
	if payload.Message != "" && message == "" {
		return nil, &bitbucket.CLIError{Message: "Merge message cannot be empty.", ExitCode: 1}
	}

	request := prMergeRequest{Type: "pullrequest_merge_parameters"}
	if strategy != "" {
		request.MergeStrategy = strategy
	}
	if message != "" {
		request.Message = message
	}

	var pr bitbucket.PullRequest
	if err := client.Post(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge", workspace, repo, id), request, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func Comments(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) ([]bitbucket.PullRequestComment, error) {
	return bitbucket.Paginate[bitbucket.PullRequestComment](ctx, client, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repo, id))
}

func AddComment(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return &bitbucket.CLIError{Message: "Comment message cannot be empty.", ExitCode: 1}
	}

	return client.Post(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repo, id), map[string]any{
		"content": map[string]string{"raw": message},
	}, nil)
}

func Commits(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) ([]bitbucket.Commit, error) {
	return bitbucket.Paginate[bitbucket.Commit](ctx, client, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/commits", workspace, repo, id))
}

func Files(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) ([]bitbucket.PullRequestDiffstat, error) {
	return bitbucket.Paginate[bitbucket.PullRequestDiffstat](ctx, client, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diffstat", workspace, repo, id))
}

func Diff(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) (string, error) {
	return client.GetText(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diff", workspace, repo, id))
}

func Tasks(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) ([]bitbucket.PullRequestTask, error) {
	return bitbucket.Paginate[bitbucket.PullRequestTask](ctx, client, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/tasks", workspace, repo, id))
}

func CreateTask(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int, message string) (*bitbucket.PullRequestTask, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, &bitbucket.CLIError{Message: "Task message cannot be empty.", ExitCode: 1}
	}

	var task bitbucket.PullRequestTask
	if err := client.Post(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/tasks", workspace, repo, id), map[string]any{
		"content": map[string]string{"raw": message},
	}, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

func ResolveTask(ctx context.Context, client *bitbucket.Client, workspace, repo string, prID, taskID int) error {
	return client.Patch(ctx, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/tasks/%d", workspace, repo, prID, taskID), map[string]string{
		"state": "RESOLVED",
	}, nil)
}

func Reviewers(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) ([]bitbucket.Account, error) {
	pr, err := Get(ctx, client, workspace, repo, id)
	if err != nil {
		return nil, err
	}

	return pr.Reviewers, nil
}

func Checks(ctx context.Context, client *bitbucket.Client, workspace, repo string, id int) ([]bitbucket.CommitStatus, error) {
	return bitbucket.Paginate[bitbucket.CommitStatus](ctx, client, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/statuses", workspace, repo, id))
}

func buildPRQuery(opts PrListOptions) string {
	author := strings.TrimSpace(opts.Author)
	reviewer := strings.TrimSpace(opts.Reviewer)
	source := strings.TrimSpace(opts.Source)
	target := strings.TrimSpace(opts.Target)
	query := strings.TrimSpace(opts.Query)

	parts := make([]string, 0, 5)
	if author != "" {
		escaped := escapePRQueryLiteral(author)
		parts = append(parts, fmt.Sprintf("(author.display_name ~ \"%s\" OR author.nickname ~ \"%s\" OR author.uuid = \"%s\")", escaped, escaped, escaped))
	}
	if reviewer != "" {
		escaped := escapePRQueryLiteral(reviewer)
		parts = append(parts, fmt.Sprintf("(reviewers.display_name ~ \"%s\" OR reviewers.nickname ~ \"%s\" OR reviewers.uuid = \"%s\")", escaped, escaped, escaped))
	}
	if source != "" {
		parts = append(parts, fmt.Sprintf("source.branch.name = \"%s\"", escapePRQueryLiteral(source)))
	}
	if target != "" {
		parts = append(parts, fmt.Sprintf("destination.branch.name = \"%s\"", escapePRQueryLiteral(target)))
	}
	if query != "" {
		parts = append(parts, query)
	}

	return strings.Join(parts, " AND ")
}

func escapePRQueryLiteral(value string) string {
	return strings.NewReplacer(`\\`, `\\\\`, `"`, `\\"`).Replace(value)
}

func resolveCreatePayload(payload PrCreatePayload) (map[string]any, error) {
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		return nil, &bitbucket.CLIError{Message: "Pull request title is required.", ExitCode: 1}
	}

	source := strings.TrimSpace(payload.SourceBranch)
	if source == "" {
		return nil, &bitbucket.CLIError{Message: "Source branch is required.", ExitCode: 1}
	}

	destination := strings.TrimSpace(payload.DestinationBranch)
	if destination == "" {
		return nil, &bitbucket.CLIError{Message: "Destination branch is required.", ExitCode: 1}
	}

	request := map[string]any{
		"title":       title,
		"source":      map[string]any{"branch": map[string]string{"name": source}},
		"destination": map[string]any{"branch": map[string]string{"name": destination}},
	}

	if description := strings.TrimSpace(payload.Description); description != "" {
		request["description"] = description
	}
	if reviewers := resolveReviewerRefs(payload.Reviewers); len(reviewers) > 0 {
		request["reviewers"] = reviewers
	}
	if payload.CloseSourceBranch {
		request["close_source_branch"] = true
	}

	return request, nil
}

func resolveUpdatePayload(payload PrUpdatePayload) (map[string]any, error) {
	request := map[string]any{}

	if payload.Title != "" {
		title := strings.TrimSpace(payload.Title)
		if title == "" {
			return nil, &bitbucket.CLIError{Message: "Pull request title cannot be empty.", ExitCode: 1}
		}
		request["title"] = title
	}

	// --description-file takes precedence over --description
	if payload.DescriptionFile != "" {
		data, err := os.ReadFile(payload.DescriptionFile)
		if err != nil {
			return nil, &bitbucket.CLIError{Message: fmt.Sprintf("Cannot read description file: %s", err), ExitCode: 1}
		}
		desc := strings.TrimSpace(string(data))
		if desc == "" {
			return nil, &bitbucket.CLIError{Message: "Description file is empty.", ExitCode: 1}
		}
		request["description"] = desc
	} else if payload.Description != "" {
		description := strings.TrimSpace(payload.Description)
		if description == "" {
			return nil, &bitbucket.CLIError{Message: "Pull request description cannot be empty.", ExitCode: 1}
		}
		request["description"] = description
	}

	if payload.DestinationBranch != "" {
		request["destination"] = map[string]any{"branch": map[string]string{"name": payload.DestinationBranch}}
	}

	if payload.Reviewers != nil {
		request["reviewers"] = resolveReviewerRefs(payload.Reviewers)
	}

	if len(request) == 0 {
		return nil, &bitbucket.CLIError{Message: "At least one field must be provided to update the pull request.", ExitCode: 1}
	}

	return request, nil
}

func resolveReviewerRefs(reviewers []string) []reviewerRef {
	resolved := make([]reviewerRef, 0, len(reviewers))
	for _, reviewer := range reviewers {
		trimmed := strings.TrimSpace(reviewer)
		if trimmed == "" {
			continue
		}

		ref := reviewerRef{}
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			ref.UUID = trimmed
		} else {
			ref.Nickname = trimmed
		}
		resolved = append(resolved, ref)
	}

	return resolved
}

func normalizeMergeStrategy(strategy string) (string, error) {
	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		return "", nil
	}

	switch strategy {
	case "merge", "merge_commit":
		return "merge_commit", nil
	case "squash":
		return "squash", nil
	case "fast_forward":
		return "fast_forward", nil
	case "squash_fast_forward":
		return "squash_fast_forward", nil
	case "rebase_fast_forward":
		return "rebase_fast_forward", nil
	case "rebase_merge":
		return "rebase_merge", nil
	default:
		return "", &bitbucket.CLIError{Message: "Invalid merge strategy. Supported strategies: merge_commit, squash, fast_forward, squash_fast_forward, rebase_fast_forward, rebase_merge.", ExitCode: 1}
	}
}

func ValidatePRListLimit(limit string) error {
	limit = strings.TrimSpace(limit)
	if limit == "" {
		return nil
	}

	value, err := strconv.Atoi(limit)
	if err != nil || value <= 0 {
		return &bitbucket.CLIError{Message: "Limit must be a positive integer.", ExitCode: 1}
	}

	return nil
}
