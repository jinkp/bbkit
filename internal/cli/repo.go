package cli

import (
	"strings"

	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/output"
	"github.com/jinkp/bbkit/internal/services"
	"github.com/spf13/cobra"
)

func NewRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage Bitbucket repositories",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List repositories in a workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			workspace, err := ResolveWorkspace(cmd, cfg)
			if err != nil {
				return err
			}

			client, err := CreateBitbucketClient(cfg)
			if err != nil {
				return err
			}

			repositories, err := services.ListRepositories(cmd.Context(), client, workspace)
			if err != nil {
				return err
			}

			if ShouldOutputJSON(cmd, cfg) {
				return output.PrintJSON(repositories)
			}

			rows := make([][]string, 0, len(repositories))
			for _, repository := range repositories {
				rows = append(rows, []string{
					repository.Name,
					repository.Slug,
					valueOrDash(repository.Language),
					formatDate(repository.UpdatedOn),
					formatPrivate(repository.IsPrivate),
				})
			}

			output.PrintTable([]string{"Name", "Slug", "Language", "Updated", "Private"}, rows)
			return nil
		},
	})

	return cmd
}

func formatDate(value string) string {
	if before, _, ok := strings.Cut(value, "T"); ok {
		return before
	}

	return valueOrDash(value)
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}

	return value
}

func formatPrivate(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}
