package services

import (
	"context"
	"errors"
	"strings"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/secrets"
)

type AuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	Source        string `json:"source,omitempty"`
}

var (
	secretsGetWithSource = secrets.GetWithSource
	secretsClear         = secrets.Clear
	secretsGet           = secrets.Get
	secretsSave          = secrets.Save
	configSave           = config.Save
)

func Status() (AuthStatus, error) {
	creds, source, err := secretsGetWithSource()
	if err != nil {
		return AuthStatus{}, err
	}

	if creds == nil {
		return AuthStatus{Authenticated: false}, nil
	}

	return AuthStatus{
		Authenticated: true,
		Username:      creds.Username,
		Source:        source,
	}, nil
}

func Logout() error {
	return secretsClear()
}

func GetCredentialsOrFail() (*secrets.Credentials, error) {
	creds, err := secretsGet()
	if err != nil {
		return nil, err
	}
	if creds == nil || creds.Username == "" || creds.APIToken == "" {
		return nil, &bitbucket.CLIError{Message: "Not authenticated. Run `bbk auth login`.", ExitCode: 1}
	}

	return creds, nil
}

func Login(ctx context.Context, client *bitbucket.Client, creds secrets.Credentials, workspace string, cfg *config.Config) error {
	if client == nil {
		return errors.New("bitbucket client is required")
	}

	var user map[string]any
	if err := client.Get(ctx, "/user", &user); err != nil {
		var cliErr *bitbucket.CLIError
		if errors.As(err, &cliErr) {
			if cliErr.ExitCode == 1 && (strings.Contains(cliErr.Message, "Authentication failed") || strings.Contains(cliErr.Message, "Permission denied")) {
				return &bitbucket.CLIError{Message: "Invalid credentials. Check your email and API token.", ExitCode: 1}
			}
		}
		return err
	}

	if err := secretsSave(creds); err != nil {
		return err
	}

	if cfg == nil {
		cfg = &config.Config{}
	}

	cfg.Username = creds.Username
	cfg.Workspace = workspace

	return configSave(cfg)
}
