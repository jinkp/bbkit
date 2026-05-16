package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/git"
	"github.com/jinkp/bbkit/internal/output"
	"github.com/jinkp/bbkit/internal/secrets"
	"github.com/jinkp/bbkit/internal/services"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Bitbucket authentication",
	}

	cmd.AddCommand(newAuthLoginCmd(), newAuthStatusCmd(), newAuthLogoutCmd())

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Bitbucket Cloud using an API token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			detectedEmail := detectEmailHint()
			workspaceDefault := detectWorkspaceDefault(cfg)

			reader := bufio.NewReader(cmd.InOrStdin())

			email, err := promptText(reader, cmd.OutOrStdout(), "Atlassian email (e.g. you@company.com)", detectedEmail, detectedEmail)
			if err != nil {
				return err
			}
			if !git.IsEmail(email) {
				return &bitbucket.CLIError{Message: "Enter a valid Atlassian email address.", ExitCode: 1}
			}

			token, err := promptPassword(cmd.OutOrStdout(), "Bitbucket API token")
			if err != nil {
				return err
			}
			if strings.TrimSpace(token) == "" {
				return &bitbucket.CLIError{Message: "API token is required.", ExitCode: 1}
			}

			workspace, err := promptText(reader, cmd.OutOrStdout(), "Default Bitbucket workspace slug (e.g. my-company)", workspaceDefault, "")
			if err != nil {
				return err
			}
			if workspace == "" {
				return &bitbucket.CLIError{Message: "Workspace is required.", ExitCode: 1}
			}

			stop := startSpinner(cmd.OutOrStdout(), "Verifying credentials")
			credentials := secrets.Credentials{Username: email, APIToken: token}
			client := bitbucket.NewClient(email, token)
			loginErr := services.Login(cmd.Context(), client, credentials, workspace, cfg)
			stop()
			if loginErr != nil {
				return loginErr
			}

			output.PrintSuccess(fmt.Sprintf("Logged in as %s", email))
			output.PrintSuccess(fmt.Sprintf("Default workspace set to: %s", workspace))
			output.PrintInfo("Tip: workspace slug is the identifier in https://bitbucket.org/{workspace}/")
			return nil
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := services.Status()
			if err != nil {
				return err
			}

			if GetAppContext(cmd).JSON {
				return output.PrintJSON(status)
			}

			if status.Authenticated {
				output.PrintSuccess(fmt.Sprintf("Authenticated as %s (via %s)", status.Username, status.Source))
				return nil
			}

			output.PrintWarning("Not authenticated. Run `bbk auth login`.")
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := services.Logout(); err != nil {
				return err
			}

			output.PrintSuccess("Logged out successfully.")
			return nil
		},
	}
}

func detectEmailHint() string {
	gitCred, err := git.GetGitCredential("bitbucket.org")
	if err == nil && gitCred != nil && git.IsEmail(gitCred.Username) {
		return gitCred.Username
	}

	gitEmail, err := git.GetGitUserEmail()
	if err == nil && git.IsEmail(gitEmail) {
		return gitEmail
	}

	return ""
}

func detectWorkspaceDefault(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Workspace) != "" {
		return strings.TrimSpace(cfg.Workspace)
	}

	remote, err := git.InferFromGitRemote()
	if err == nil && remote != nil {
		return strings.TrimSpace(remote.Workspace)
	}

	return ""
}

func promptText(reader *bufio.Reader, out io.Writer, label, defaultValue, hint string) (string, error) {
	if hint != "" {
		_, _ = fmt.Fprintf(out, "Detected: %s\n", hint)
	}

	if strings.TrimSpace(defaultValue) != "" {
		_, _ = fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
	} else {
		_, _ = fmt.Fprintf(out, "%s: ", label)
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		value = strings.TrimSpace(defaultValue)
	}

	return value, nil
}

func promptPassword(out io.Writer, label string) (string, error) {
	_, _ = fmt.Fprintf(out, "%s: ", label)
	value, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(out)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func startSpinner(out io.Writer, message string) func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	var once sync.Once
	done := make(chan struct{})

	go func() {
		i := 0
		for {
			select {
			case <-done:
				_, _ = fmt.Fprintf(out, "\r%s... done\n", message)
				return
			default:
				_, _ = fmt.Fprintf(out, "\r%s %s...", frames[i%len(frames)], message)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	return func() {
		once.Do(func() { close(done) })
		time.Sleep(10 * time.Millisecond) // let goroutine print final state
	}
}
