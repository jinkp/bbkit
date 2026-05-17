package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/version"
	"github.com/spf13/cobra"
)

type AppContext struct {
	JSON      bool
	Workspace string
	Repo      string
}

type appContextKey struct{}

func Execute() error {
	err := NewRootCmd().Execute()
	if err == nil {
		return nil
	}

	var cliErr *bitbucket.CLIError
	if errors.As(err, &cliErr) {
		_, _ = fmt.Fprintln(os.Stderr, cliErr.Message)
		os.Exit(cliErr.ExitCode)
	}

	_, _ = fmt.Fprintln(os.Stderr, "An unexpected error occurred.")
	if os.Getenv("DEBUG") != "" {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(2)

	return err
}

func NewRootCmd() *cobra.Command {
	appCtx := AppContext{}

	cmd := &cobra.Command{
		Use:           "bbk",
		Short:         "A modern CLI for Bitbucket Cloud",
		Long:          "A modern CLI for Bitbucket Cloud",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.WithValue(cmd.Context(), appContextKey{}, AppContext{
				JSON:      appCtx.JSON,
				Workspace: appCtx.Workspace,
				Repo:      appCtx.Repo,
			})
			cmd.SetContext(ctx)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&appCtx.JSON, "json", false, "Output JSON")
	cmd.PersistentFlags().StringVar(&appCtx.Workspace, "workspace", "", "Bitbucket workspace slug")
	cmd.PersistentFlags().StringVar(&appCtx.Repo, "repo", "", "Bitbucket repository slug")

	cmd.AddCommand(
		NewVersionCmd(),
		NewConfigCmd(),
		NewAuthCmd(),
		NewSetupCmd(),
		NewRepoCmd(),
		NewBranchCmd(),
		NewPipelineCmd(),
		NewPRCmd(),
		NewTUICmd(),
		NewMCPCmd(),
	)

	return cmd
}
