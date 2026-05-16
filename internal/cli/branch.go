package cli

import (
	"strconv"
	"strings"
	"time"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/output"
	"github.com/jinkp/bbkit/internal/services"
	"github.com/spf13/cobra"
)

func NewBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage Bitbucket branches",
	}

	cmd.AddCommand(newBranchListCmd(), newBranchStaleCmd())

	return cmd
}

func newBranchListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List repository branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBranchTableCommand(cmd, func(clientCtx branchCommandContext) ([]bitbucket.Branch, error) {
				return services.ListBranches(cmd.Context(), clientCtx.client, clientCtx.workspace, clientCtx.repo)
			})
		},
	}
}

func newBranchStaleCmd() *cobra.Command {
	var days string

	cmd := &cobra.Command{
		Use:   "stale",
		Short: "List stale branches older than N days",
		RunE: func(cmd *cobra.Command, args []string) error {
			parsedDays, err := strconv.Atoi(days)
			if err != nil || parsedDays < 0 {
				return &bitbucket.CLIError{Message: "The --days option must be a non-negative number.", ExitCode: 1}
			}

			return runBranchTableCommand(cmd, func(clientCtx branchCommandContext) ([]bitbucket.Branch, error) {
				return services.StaleBranches(cmd.Context(), clientCtx.client, clientCtx.workspace, clientCtx.repo, parsedDays)
			})
		},
	}

	cmd.Flags().StringVar(&days, "days", "30", "Minimum branch age in days")

	return cmd
}

type branchCommandContext struct {
	client    *bitbucket.Client
	workspace string
	repo      string
	cfg       *config.Config
}

func runBranchTableCommand(cmd *cobra.Command, fetch func(branchCommandContext) ([]bitbucket.Branch, error)) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	workspace, repo, err := ResolveWorkspaceRepo(cmd, cfg)
	if err != nil {
		return err
	}

	client, err := CreateBitbucketClient(cfg)
	if err != nil {
		return err
	}

	branches, err := fetch(branchCommandContext{client: client, workspace: workspace, repo: repo, cfg: cfg})
	if err != nil {
		return err
	}

	if ShouldOutputJSON(cmd, cfg) {
		return output.PrintJSON(branches)
	}

	rows := make([][]string, 0, len(branches))
	for _, branch := range branches {
		rows = append(rows, []string{
			branch.Name,
			shortHash(branch.Target.Hash),
			branchAuthor(branch),
			formatBranchDate(branch.Target.Date),
		})
	}

	output.PrintTable([]string{"Name", "Hash", "Author", "Date"}, rows)
	return nil
}

func shortHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}

	return hash[:7]
}

func branchAuthor(branch bitbucket.Branch) string {
	return valueOrDash(branch.Target.Author.User.DisplayName)
}

func formatBranchDate(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.Format("2006-01-02")
	}

	if before, _, ok := strings.Cut(value, "T"); ok {
		return before
	}

	return valueOrDash(value)
}
