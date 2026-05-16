package cli

import (
	"fmt"

	"github.com/jinkp/bbkit/internal/output"
	"github.com/jinkp/bbkit/internal/version"
	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the installed bbkit CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetAppContext(cmd).JSON {
				return output.PrintJSON(map[string]string{"version": version.Version})
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "bbk version %s\n", version.Version)
			return err
		},
	}
}
