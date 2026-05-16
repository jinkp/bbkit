package cli

import (
	"fmt"
	"sort"

	"github.com/jinkp/bbkit/internal/bitbucket"
	configpkg "github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/output"
	"github.com/spf13/cobra"
)

var validConfigKeys = map[string]struct{}{
	"workspace":     {},
	"username":      {},
	"defaultOutput": {},
}

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and manage bbkit configuration",
	}

	cmd.AddCommand(newConfigListCmd(), newConfigGetCmd(), newConfigSetCmd())

	return cmd
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configpkg.Load()
			if err != nil {
				return err
			}

			entries := effectiveConfigEntries(cfg)
			if len(entries) == 0 {
				output.PrintWarning("No configuration set. Run `bbk setup` to get started.")
				return nil
			}

			keys := make([]string, 0, len(entries))
			for key := range entries {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			for _, key := range keys {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", key, entries[key]); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if err := validateConfigKey(key); err != nil {
				return err
			}

			cfg, err := configpkg.Load()
			if err != nil {
				return err
			}

			value := effectiveConfigEntries(cfg)[key]
			if value == "" {
				return &bitbucket.CLIError{Message: fmt.Sprintf("%s is not set", key), ExitCode: 1}
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), value)
			return err
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			if err := validateConfigKey(key); err != nil {
				return err
			}
			if key == "defaultOutput" && value != "table" && value != "json" {
				return &bitbucket.CLIError{Message: "defaultOutput must be \"table\" or \"json\"", ExitCode: 1}
			}

			cfg, err := configpkg.Load()
			if err != nil {
				return err
			}

			switch key {
			case "workspace":
				cfg.Workspace = value
			case "username":
				cfg.Username = value
			case "defaultOutput":
				cfg.DefaultOutput = value
			}

			if err := configpkg.Save(cfg); err != nil {
				return err
			}

			output.PrintSuccess(fmt.Sprintf("%s = %s", key, value))
			return nil
		},
	}
}

func validateConfigKey(key string) error {
	if _, ok := validConfigKeys[key]; ok {
		return nil
	}

	return &bitbucket.CLIError{Message: fmt.Sprintf("Unknown config key: %s. Valid keys: workspace, username, defaultOutput", key), ExitCode: 1}
}

func effectiveConfigEntries(cfg *configpkg.Config) map[string]string {
	entries := map[string]string{}
	if cfg == nil {
		cfg = &configpkg.Config{}
	}

	if cfg.Workspace != "" {
		entries["workspace"] = cfg.Workspace
	}
	if cfg.Username != "" {
		entries["username"] = cfg.Username
	}
	if cfg.DefaultOutput != "" {
		entries["defaultOutput"] = cfg.DefaultOutput
	}

	if workspace, err := configpkg.GetWorkspace(); err == nil && workspace != "" {
		entries["workspace"] = workspace
	}
	if username, err := configpkg.GetUsername(); err == nil && username != "" {
		entries["username"] = username
	}

	return entries
}
