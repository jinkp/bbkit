package cli

import (
	"log"
	"os"

	bbmcp "github.com/jinkp/bbkit/internal/mcp"
	"github.com/spf13/cobra"
)

// NewMCPCmd returns the `bbk mcp` command.
//
// CRITICAL: This command MUST NOT write ANYTHING to stdout before calling
// mcp.StartServer(). The MCP stdio transport owns stdout entirely.
// Use os.Stderr for any diagnostic output.
func NewMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "mcp",
		Short:        "Start the bbkit MCP stdio server (for use with opencode, Claude, etc.)",
		SilenceUsage: true,
		// Disable cobra's built-in usage/error printing to stdout.
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Redirect default logger to stderr — belt-and-suspenders guard
			// against any imported package writing to the default log sink.
			log.SetOutput(os.Stderr)

			// StartServer blocks until the client disconnects or a signal is received.
			return bbmcp.StartServer()
		},
	}
}
