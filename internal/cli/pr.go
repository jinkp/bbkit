package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/output"
	"github.com/jinkp/bbkit/internal/services"
	"github.com/spf13/cobra"
)

func NewPRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage Bitbucket pull requests",
	}

	cmd.AddCommand(
		newPRListCmd(),
		newPRViewCmd(),
		newPRStatusCmd(),
		newPRCommitsCmd(),
		newPRReviewersCmd(),
		newPRTasksCmd(),
		newPRChecksCmd(),
		newPRCreateCmd(),
		newPRUpdateCmd(),
		newPRCheckoutCmd(),
		newPROpenCmd(),
		newPRApproveCmd(),
		newPRDeclineCmd(),
		newPRMergeCmd(),
		newPRCommentsCmd(),
		newPRCommentCmd(),
		newPRFilesCmd(),
		newPRDiffCmd(),
		newPRTaskCmd(),
	)

	return cmd
}

type prCommandContext struct {
	client    *bitbucket.Client
	workspace string
	repo      string
	cfg       *config.Config
	json      bool
}

func loadPRContext(cmd *cobra.Command) (*prCommandContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	workspace, repo, err := ResolveWorkspaceRepo(cmd, cfg)
	if err != nil {
		return nil, err
	}

	client, err := CreateBitbucketClient(cfg)
	if err != nil {
		return nil, err
	}

	return &prCommandContext{
		client:    client,
		workspace: workspace,
		repo:      repo,
		cfg:       cfg,
		json:      ShouldOutputJSON(cmd, cfg),
	}, nil
}

func newPRListCmd() *cobra.Command {
	var states []string
	var all bool
	var author string
	var reviewer string
	var source string
	var target string
	var query string
	var limit string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := services.ValidatePRListLimit(limit); err != nil {
				return err
			}

			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}

			pullRequests, err := services.List(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, services.PrListOptions{
				States:   states,
				All:      all,
				Author:   author,
				Reviewer: reviewer,
				Source:   source,
				Target:   target,
				Query:    query,
				Limit:    limit,
			})
			if err != nil {
				return err
			}

			if ctx.json {
				return output.PrintJSON(pullRequests)
			}

			rows := make([][]string, 0, len(pullRequests))
			for _, pr := range pullRequests {
				rows = append(rows, []string{
					strconv.Itoa(pr.ID),
					pr.Title,
					formatPRAccount(&pr.Author),
					fmt.Sprintf("%s → %s", pr.Source.Branch.Name, pr.Destination.Branch.Name),
					string(pr.State),
					formatPRDateTime(pr.CreatedOn),
				})
			}

			output.PrintTable([]string{"ID", "Title", "Author", "Source→Dest", "State", "Created"}, rows)
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&states, "state", nil, "Filter by pull request state (repeatable: OPEN, MERGED, DECLINED, SUPERSEDED)")
	cmd.Flags().BoolVar(&all, "all", false, "Include all states (cannot be combined with --state)")
	cmd.Flags().StringVar(&author, "author", "", "Filter by author (UUID preferred)")
	cmd.Flags().StringVar(&reviewer, "reviewer", "", "Filter by reviewer (UUID preferred)")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source branch")
	cmd.Flags().StringVar(&target, "target", "", "Filter by target branch")
	cmd.Flags().StringVar(&query, "query", "", "Bitbucket API query filter")
	cmd.Flags().StringVar(&limit, "limit", "", "Maximum results per page")

	return cmd
}

func newPRViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <id>",
		Short: "Show pull request details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}

			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}

			pr, err := services.Get(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}

			if ctx.json {
				return output.PrintJSON(pr)
			}

			output.PrintTable([]string{"Field", "Value"}, [][]string{
				{"ID", strconv.Itoa(pr.ID)},
				{"Title", pr.Title},
				{"Description", valueOrDash(pr.Description)},
				{"State", string(pr.State)},
				{"Author", formatPRAccount(&pr.Author)},
				{"Branches", fmt.Sprintf("%s → %s", pr.Source.Branch.Name, pr.Destination.Branch.Name)},
				{"Reviewers", strings.Join(formatPRAccounts(pr.Reviewers), ", ")},
				{"Participants", strings.Join(formatParticipants(pr.Participants), ", ")},
				{"Created", formatPRDateTime(pr.CreatedOn)},
				{"Updated", formatPRDateTime(pr.UpdatedOn)},
				{"URL", pr.Links.HTML.Href},
			})
			return nil
		},
	}
}

func newPRStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "Show pull request state and merge check status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}

			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}

			pr, err := services.Get(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			checks, err := services.Checks(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}

			summary := map[string]any{
				"pull_request": pr,
				"merge_check":  summarizeMergeChecks(checks),
				"checks":       checks,
			}
			if ctx.json {
				return output.PrintJSON(summary)
			}

			output.PrintTable([]string{"Field", "Value"}, [][]string{
				{"State", string(pr.State)},
				{"Merge Check", summarizeMergeChecks(checks)},
				{"Branches", fmt.Sprintf("%s → %s", pr.Source.Branch.Name, pr.Destination.Branch.Name)},
				{"Tasks", intString(pr.TaskCount)},
				{"Comments", intString(pr.CommentCount)},
				{"Approvals", approvalSummary(pr)},
				{"Updated", formatPRDateTime(pr.UpdatedOn)},
			})
			return nil
		},
	}
}

func newPRCommitsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commits <id>",
		Short: "List commits on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			commits, err := services.Commits(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(commits)
			}
			rows := make([][]string, 0, len(commits))
			for _, commit := range commits {
				rows = append(rows, []string{
					shortPRHash(commit.Hash),
					firstPRLine(commit.Message),
					formatCommitAuthor(commit),
					formatPRDateTime(commit.Date),
					commitURL(commit),
				})
			}
			output.PrintTable([]string{"Hash", "Message", "Author", "Date", "URL"}, rows)
			return nil
		},
	}
}

func newPRReviewersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reviewers <id>",
		Short: "List reviewers with approval status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			pr, err := services.Get(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(map[string]any{"reviewers": pr.Reviewers, "participants": pr.Participants})
			}
			rows := make([][]string, 0, len(pr.Reviewers)+len(pr.Participants))
			for _, reviewer := range pr.Reviewers {
				rows = append(rows, []string{formatPRAccount(&reviewer), "reviewer", "", ""})
			}
			for _, participant := range pr.Participants {
				approved := ""
				if participant.Approved {
					approved = "yes"
				} else if participant.User != nil {
					approved = "no"
				}
				rows = append(rows, []string{formatPRAccount(participant.User), valueOrDash(participant.Role), approved, valueOrDash(participant.State)})
			}
			output.PrintTable([]string{"Name", "Role", "Approved", "State"}, rows)
			return nil
		},
	}
}

func newPRTasksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tasks <id>",
		Short: "List tasks on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			tasks, err := services.Tasks(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(tasks)
			}
			rows := make([][]string, 0, len(tasks))
			for _, task := range tasks {
				rows = append(rows, []string{
					strconv.Itoa(task.ID),
					valueOrDash(task.State),
					prTaskContent(task),
					formatPRAccount(task.Creator),
					formatPRAccount(task.Assignee),
					formatPRDateTime(task.CreatedOn),
					formatPRDateTime(task.UpdatedOn),
				})
			}
			output.PrintTable([]string{"ID", "State", "Content", "Creator", "Assignee", "Created", "Updated"}, rows)
			return nil
		},
	}
}

func newPRTaskCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "task", Short: "Manage pull request tasks"}
	cmd.AddCommand(newPRTaskCreateCmd(), newPRTaskResolveCmd())
	return cmd
}

func newPRTaskCreateCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "create <id>",
		Short: "Create a pull request task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			task, err := services.CreateTask(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID, message)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Task #%d created on PR #%d.\n", task.ID, prID)
			return err
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "Task message")
	_ = cmd.MarkFlagRequired("message")
	return cmd
}

func newPRTaskResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <id> <taskId>",
		Short: "Resolve a pull request task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			resolvedTaskID, err := strconv.Atoi(strings.TrimSpace(args[1]))
			if err != nil || resolvedTaskID <= 0 {
				return &bitbucket.CLIError{Message: fmt.Sprintf("Invalid task ID: %s", args[1]), ExitCode: 1}
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			if err := services.ResolveTask(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID, resolvedTaskID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Task #%d resolved on PR #%d.\n", resolvedTaskID, prID)
			return err
		},
	}
}

func newPRChecksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checks <id>",
		Short: "List build status checks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			checks, err := services.Checks(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(checks)
			}
			rows := make([][]string, 0, len(checks))
			for _, check := range checks {
				rows = append(rows, []string{check.Key, valueOrDash(check.Name), valueOrDash(check.State), valueOrDash(check.Description), formatPRDateTime(check.UpdatedOn), valueOrDash(check.URL)})
			}
			output.PrintTable([]string{"Key", "Name", "State", "Description", "Updated", "URL"}, rows)
			return nil
		},
	}
}

func newPRCreateCmd() *cobra.Command {
	var title string
	var description string
	var source string
	var destination string
	var reviewers string
	var closeSourceBranch bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			pr, err := services.Create(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, services.PrCreatePayload{
				Title:             title,
				Description:       description,
				SourceBranch:      source,
				DestinationBranch: destination,
				Reviewers:         parseReviewersFlag(reviewers),
				CloseSourceBranch: closeSourceBranch,
			})
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(pr)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "PR created: %s\n", pr.Links.HTML.Href)
			return err
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Pull request title")
	cmd.Flags().StringVar(&description, "description", "", "Pull request description")
	cmd.Flags().StringVar(&source, "source", "", "Source branch")
	cmd.Flags().StringVar(&destination, "destination", "", "Destination branch")
	cmd.Flags().StringVar(&reviewers, "reviewers", "", "Comma-separated reviewer identifiers")
	cmd.Flags().BoolVar(&closeSourceBranch, "close-source-branch", false, "Close the source branch after merge")
	return cmd
}

func newPRUpdateCmd() *cobra.Command {
	var title string
	var description string
	var descriptionFile string
	var target string
	var reviewers string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			payload := services.PrUpdatePayload{
				Title:             title,
				Description:       description,
				DescriptionFile:   descriptionFile,
				DestinationBranch: target,
			}
			if flag := cmd.Flags().Lookup("reviewers"); flag != nil && flag.Changed {
				payload.Reviewers = parseReviewersFlag(reviewers)
			}
			pr, err := services.Update(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID, payload)
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(pr)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "PR #%d updated: %s\n", pr.ID, pr.Links.HTML.Href)
			return err
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "New pull request title")
	cmd.Flags().StringVar(&description, "description", "", "New pull request description")
	cmd.Flags().StringVar(&descriptionFile, "description-file", "", "Read description from file")
	cmd.Flags().StringVar(&target, "target", "", "New destination branch")
	cmd.Flags().StringVar(&reviewers, "reviewers", "", "Comma-separated reviewer identifiers")
	return cmd
}

func newPRCheckoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checkout <id>",
		Short: "Check out a pull request source branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			pr, err := services.Get(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if err := services.CheckoutSourceBranch(cmd.Context(), pr.Source.Branch.Name); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Checked out PR #%d source branch: %s\n", pr.ID, pr.Source.Branch.Name)
			return err
		},
	}
}

func newPROpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <id>",
		Short: "Open a pull request in the browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			pr, err := services.Get(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if err := services.OpenURL(pr.Links.HTML.Href); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Opened PR #%d: %s\n", pr.ID, pr.Links.HTML.Href)
			return err
		},
	}
}

func newPRApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			if err := services.Approve(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Approved PR #%d.\n", prID)
			return err
		},
	}
}

func newPRDeclineCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "decline <id>",
		Short: "Decline a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			if reason != "" {
				if err := services.AddComment(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID, reason); err != nil {
					return err
				}
			}
			if err := services.Decline(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Declined PR #%d.\n", prID)
			return err
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Post a comment with the decline reason before declining")
	return cmd
}

func newPRMergeCmd() *cobra.Command {
	var yes bool
	var strategy string
	var message string

	cmd := &cobra.Command{
		Use:   "merge <id>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}

			if !yes {
				confirmed, err := confirmPRMerge(cmd, prID, ctx.workspace, ctx.repo)
				if err != nil {
					return err
				}
				if !confirmed {
					_, err = fmt.Fprintf(cmd.OutOrStdout(), "Merge cancelled. PR #%d was not merged.\n", prID)
					return err
				}
			}

			pr, err := services.Merge(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID, services.PrMergePayload{Strategy: strategy, Message: message})
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(pr)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Merged PR #%d: %s\n", pr.ID, pr.Links.HTML.Href)
			return err
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip merge confirmation")
	cmd.Flags().StringVar(&strategy, "strategy", "", "Merge strategy: merge_commit, squash, fast_forward, squash_fast_forward, rebase_fast_forward, rebase_merge")
	cmd.Flags().StringVar(&message, "message", "", "Merge commit message")
	return cmd
}

func newPRCommentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "comments <id>",
		Short: "List pull request comments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			comments, err := services.Comments(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(comments)
			}
			rows := make([][]string, 0, len(comments))
			for _, comment := range comments {
				rows = append(rows, []string{strconv.Itoa(comment.ID), commentAuthor(comment), formatPRDateTime(comment.CreatedOn), commentRaw(comment), commentHTML(comment)})
			}
			output.PrintTable([]string{"ID", "Author", "Created", "Raw", "Rendered"}, rows)
			return nil
		},
	}
}

func newPRCommentCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "comment <id>",
		Short: "Post a comment on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			if err := services.AddComment(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID, message); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Comment posted on PR #%d.\n", prID)
			return err
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "Comment message")
	_ = cmd.MarkFlagRequired("message")
	return cmd
}

func newPRFilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "files <id>",
		Short: "List files changed in a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			files, err := services.Files(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			if ctx.json {
				return output.PrintJSON(files)
			}
			rows := make([][]string, 0, len(files))
			for _, file := range files {
				rows = append(rows, []string{valueOrDash(file.Status), valueOrDash(file.Type), diffOldPath(file), diffNewPath(file), intString(file.LinesAdded), intString(file.LinesRemoved)})
			}
			output.PrintTable([]string{"Status", "Type", "Old Path", "Path", "Added", "Removed"}, rows)
			return nil
		},
	}
}

func newPRDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <id>",
		Short: "Show the raw pull request diff",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ctx, err := loadPRContext(cmd)
			if err != nil {
				return err
			}
			diff, err := services.Diff(cmd.Context(), ctx.client, ctx.workspace, ctx.repo, prID)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), diff)
			return err
		},
	}
}

func parsePRID(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, &bitbucket.CLIError{Message: fmt.Sprintf("Invalid pull request ID: %s", value), ExitCode: 1}
	}
	return parsed, nil
}

func parseReviewersFlag(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func formatPRAccount(account *bitbucket.Account) string {
	if account == nil {
		return "-"
	}
	if strings.TrimSpace(account.DisplayName) != "" {
		return account.DisplayName
	}
	if strings.TrimSpace(account.Nickname) != "" {
		return account.Nickname
	}
	if strings.TrimSpace(account.UUID) != "" {
		return account.UUID
	}
	return "-"
}

func formatPRAccounts(accounts []bitbucket.Account) []string {
	if len(accounts) == 0 {
		return nil
	}
	result := make([]string, 0, len(accounts))
	for i := range accounts {
		result = append(result, formatPRAccount(&accounts[i]))
	}
	return result
}

func formatParticipants(participants []bitbucket.PullRequestParticipant) []string {
	if len(participants) == 0 {
		return nil
	}
	result := make([]string, 0, len(participants))
	for _, participant := range participants {
		result = append(result, formatPRAccount(participant.User))
	}
	return result
}

func formatPRDateTime(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err == nil {
		return parsed.Format("2006-01-02")
	}
	if before, _, ok := strings.Cut(trimmed, "T"); ok {
		return before
	}
	return trimmed
}

func shortPRHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}

func firstPRLine(value string) string {
	if value == "" {
		return "-"
	}
	if first, _, ok := strings.Cut(value, "\n"); ok {
		return first
	}
	return value
}

func formatCommitAuthor(commit bitbucket.Commit) string {
	if commit.Author == nil {
		return "-"
	}
	if commit.Author.User != nil {
		return formatPRAccount(commit.Author.User)
	}
	return valueOrDash(commit.Author.Raw)
}

func commitURL(commit bitbucket.Commit) string {
	if commit.Links == nil || commit.Links.HTML == nil {
		return "-"
	}
	return valueOrDash(commit.Links.HTML.Href)
}

func prTaskContent(task bitbucket.PullRequestTask) string {
	if task.Content == nil {
		return "-"
	}
	return valueOrDash(task.Content.Raw)
}

func commentAuthor(comment bitbucket.PullRequestComment) string {
	if comment.User == nil {
		return "-"
	}
	return valueOrDash(comment.User.DisplayName)
}

func commentRaw(comment bitbucket.PullRequestComment) string {
	if comment.Content == nil {
		return "-"
	}
	return valueOrDash(comment.Content.Raw)
}

func commentHTML(comment bitbucket.PullRequestComment) string {
	if comment.Content == nil {
		return "-"
	}
	return valueOrDash(comment.Content.HTML)
}

func diffOldPath(file bitbucket.PullRequestDiffstat) string {
	if file.Old == nil {
		return "-"
	}
	return valueOrDash(file.Old.Path)
}

func diffNewPath(file bitbucket.PullRequestDiffstat) string {
	if file.New != nil && strings.TrimSpace(file.New.Path) != "" {
		return file.New.Path
	}
	if file.Old != nil {
		return valueOrDash(file.Old.Path)
	}
	return "-"
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	return strconv.Itoa(value)
}

func approvalSummary(pr *bitbucket.PullRequest) string {
	if pr == nil {
		return "0/0"
	}
	approved := 0
	for _, participant := range pr.Participants {
		if participant.Approved {
			approved++
		}
	}
	return fmt.Sprintf("%d/%d", approved, len(pr.Participants))
}

func summarizeMergeChecks(checks []bitbucket.CommitStatus) string {
	if len(checks) == 0 {
		return "unknown"
	}

	hasPending := false
	for _, check := range checks {
		switch strings.ToUpper(strings.TrimSpace(check.State)) {
		case "SUCCESSFUL", "SUCCESS":
			continue
		case "FAILED", "FAILURE", "ERROR", "STOPPED":
			return "failed"
		default:
			hasPending = true
		}
	}

	if hasPending {
		return "pending"
	}
	return "passing"
}

func confirmPRMerge(cmd *cobra.Command, prID int, workspace, repo string) (bool, error) {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Merge PR #%d into %s/%s? [y/N]: ", prID, workspace, repo)
	if err != nil {
		return false, err
	}

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}
