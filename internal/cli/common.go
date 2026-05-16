package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/git"
	"github.com/jinkp/bbkit/internal/secrets"
	"github.com/spf13/cobra"
)

const envRepo = "BITBUCKET_REPO"

func ResolveWorkspace(cmd *cobra.Command, cfg *config.Config) (string, error) {
	if cfg == nil {
		cfg = &config.Config{}
	}

	appCtx := GetAppContext(cmd)
	workspace := firstNonEmpty(appCtx.Workspace, os.Getenv("BITBUCKET_WORKSPACE"), cfg.Workspace)

	if workspace == "" {
		remote, _ := git.InferFromGitRemote()
		if remote != nil {
			workspace = remote.Workspace
		}
	}

	if workspace == "" {
		return "", &bitbucket.CLIError{
			Message: strings.Join([]string{
				"No workspace configured.",
				"Options:",
				"  1. Use the --workspace <slug> flag",
				"  2. Set the BITBUCKET_WORKSPACE environment variable",
				"  3. Save workspace in bbkit config",
				"  4. Run from inside a git repo with a Bitbucket origin remote",
				"",
				"Your workspace slug is the identifier in your Bitbucket URL:",
				"  https://bitbucket.org/{workspace}/...",
			}, "\n"),
			ExitCode: 1,
		}
	}

	return workspace, nil
}

func ResolveWorkspaceRepo(cmd *cobra.Command, cfg *config.Config) (workspace, repo string, err error) {
	if cfg == nil {
		cfg = &config.Config{}
	}

	appCtx := GetAppContext(cmd)
	workspace = firstNonEmpty(appCtx.Workspace, os.Getenv("BITBUCKET_WORKSPACE"), cfg.Workspace)
	repo = firstNonEmpty(appCtx.Repo, os.Getenv(envRepo), cfg.Repo)

	remote, remoteErr := git.InferFromGitRemote()
	if remote != nil {
		if workspace == "" {
			workspace = remote.Workspace
		}
		if repo == "" {
			repo = remote.RepoSlug
		}
	}

	if workspace == "" {
		return "", "", &bitbucket.CLIError{
			Message: strings.Join([]string{
				"No workspace configured.",
				"Options:",
				"  1. Use the --workspace <slug> flag",
				"  2. Set the BITBUCKET_WORKSPACE environment variable",
				"  3. Save workspace in bbkit config",
				"  4. Run from inside a git repo with a Bitbucket origin remote",
				"",
				"Your workspace slug is the identifier in your Bitbucket URL:",
				"  https://bitbucket.org/{workspace}/...",
			}, "\n"),
			ExitCode: 1,
		}
	}

	if repo == "" {
		message := strings.Join([]string{
			"No repository found.",
			"Options:",
			"  1. Use the --repo <slug> flag",
			"  2. Set the BITBUCKET_REPO environment variable",
			"  3. Save repo in bbkit config",
			"  4. Run from inside a git repo with a Bitbucket origin remote",
		}, "\n")
		if remoteErr != nil {
			message += fmt.Sprintf("\n\nGit remote detection failed: %v", remoteErr)
		}

		return "", "", &bitbucket.CLIError{Message: message, ExitCode: 1}
	}

	return workspace, repo, nil
}

func CreateBitbucketClient(cfg *config.Config) (*bitbucket.Client, error) {
	if cfg == nil {
		loadedCfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		cfg = loadedCfg
	}

	creds, err := secrets.Get()
	if err != nil {
		return nil, err
	}
	if creds == nil || creds.Username == "" || creds.APIToken == "" {
		message := "Not authenticated. Run `bbk auth login` or set BITBUCKET_USERNAME and BITBUCKET_API_TOKEN."
		if cfg != nil && cfg.Username != "" {
			message = fmt.Sprintf("No credentials found for %s. Run `bbk auth login` or set BITBUCKET_USERNAME and BITBUCKET_API_TOKEN.", cfg.Username)
		}

		return nil, &bitbucket.CLIError{Message: message, ExitCode: 1}
	}

	return bitbucket.NewClient(creds.Username, creds.APIToken), nil
}

func ShouldOutputJSON(cmd *cobra.Command, cfg *config.Config) bool {
	if GetAppContext(cmd).JSON {
		return true
	}

	return cfg != nil && cfg.DefaultOutput == "json"
}

func GetAppContext(cmd *cobra.Command) AppContext {
	if cmd == nil {
		return AppContext{}
	}

	if ctx := cmd.Context(); ctx != nil {
		if appCtx, ok := ctx.Value(appContextKey{}).(AppContext); ok {
			return appCtx
		}
	}

	appCtx := AppContext{}
	appCtx.JSON, _ = cmd.Flags().GetBool("json")
	appCtx.Workspace, _ = cmd.Flags().GetString("workspace")
	appCtx.Repo, _ = cmd.Flags().GetString("repo")

	return appCtx
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
