package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/secrets"
	"github.com/jinkp/bbkit/internal/tui"
	"github.com/spf13/cobra"
)

func NewSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard for bbkit",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			model := tui.NewModel(cfg, func(result tui.SetupResult) error {
				nextCfg := *cfg
				nextCfg.Username = result.Email
				nextCfg.Workspace = result.Workspace
				nextCfg.DefaultOutput = result.OutputFormat

				if err := secrets.Save(secrets.Credentials{Username: result.Email, APIToken: result.Token}); err != nil {
					return err
				}

				return config.Save(&nextCfg)
			})

			program := tea.NewProgram(model, tea.WithAltScreen())
			finalModel, err := program.Run()
			if err != nil {
				return err
			}

			wizard, ok := finalModel.(tui.Model)
			if !ok {
				return &bitbucket.CLIError{Message: "Setup did not complete correctly.", ExitCode: 2}
			}

			if wizard.Cancelled() {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), wizard.CancelMessage())
				return nil
			}

			if wizard.Error() {
				return &bitbucket.CLIError{Message: wizard.ErrorMessage(), ExitCode: 1}
			}

			if wizard.Done() {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), wizard.DoneMessage())
			}

			return nil
		},
	}
}
