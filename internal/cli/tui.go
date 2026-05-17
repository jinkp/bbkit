package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/bbkit/internal/tui"
	"github.com/spf13/cobra"
)

func NewTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive menu for common bbkit operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			model := tui.NewMenuModel()
			p := tea.NewProgram(model, tea.WithAltScreen())

			finalModel, err := p.Run()
			if err != nil {
				return err
			}

			result := finalModel.(tui.MenuModel).Result()
			if result.Cancelled {
				return nil
			}

			return runCommand(result.Command)
		},
	}
}

func runCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find bbk executable: %w", err)
	}

	c := exec.Command(exe, parts...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}
