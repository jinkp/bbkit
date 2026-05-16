package cli

import (
	"fmt"
	"time"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/output"
	"github.com/jinkp/bbkit/internal/services"
	"github.com/spf13/cobra"
)

func NewPipelineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Manage Bitbucket pipelines",
	}

	cmd.AddCommand(newPipelineListCmd(), newPipelineRunCmd())

	return cmd
}

func newPipelineListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List recent pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			pipelines, err := services.ListPipelines(cmd.Context(), client, workspace, repo)
			if err != nil {
				return err
			}

			if ShouldOutputJSON(cmd, cfg) {
				return output.PrintJSON(pipelines)
			}

			rows := make([][]string, 0, len(pipelines))
			for _, pipeline := range pipelines {
				rows = append(rows, []string{
					fmt.Sprintf("%d", pipeline.BuildNumber),
					valueOrDash(pipeline.State.Name),
					pipelineResult(pipeline),
					valueOrDash(pipeline.Target.RefName),
					formatDuration(pipeline.DurationInSeconds),
					formatPipelineCreated(pipeline.CreatedOn),
				})
			}

			output.PrintTable([]string{"#", "State", "Result", "Branch", "Duration", "Created"}, rows)
			return nil
		},
	}
}

func newPipelineRunCmd() *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a pipeline for a branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if branch == "" {
				return &bitbucket.CLIError{Message: "required flag(s) \"branch\" not set", ExitCode: 1}
			}

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

			pipeline, err := services.RunPipeline(cmd.Context(), client, workspace, repo, branch)
			if err != nil {
				return err
			}

			if ShouldOutputJSON(cmd, cfg) {
				return output.PrintJSON(pipeline)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Pipeline #%d %s\n", pipeline.BuildNumber, valueOrDash(pipeline.State.Name))
			return err
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "Branch name to build")

	return cmd
}

func pipelineResult(pipeline bitbucket.Pipeline) string {
	if pipeline.State.Result == nil {
		return "-"
	}

	return valueOrDash(pipeline.State.Result.Name)
}

func formatDuration(seconds *int) string {
	if seconds == nil {
		return "-"
	}

	minutes := *seconds / 60
	remainingSeconds := *seconds % 60
	return fmt.Sprintf("%dm %ds", minutes, remainingSeconds)
}

func formatPipelineCreated(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return valueOrDash(value)
	}

	return parsed.Format("2006-01-02 15:04")
}
