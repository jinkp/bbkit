package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		url        string
		want       string
	}{
		{name: "401", statusCode: 401, want: "Authentication failed. Run `bbk auth login`."},
		{name: "403", statusCode: 403, want: "Permission denied. Check your API token scopes."},
		{name: "404", statusCode: 404, url: "https://api.bitbucket.org/2.0/repositories/ws/repo", want: "Resource not found: https://api.bitbucket.org/2.0/repositories/ws/repo."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MapAPIError(tt.statusCode, tt.url, "body")
			var cliErr *CLIError
			require.ErrorAs(t, err, &cliErr)
			require.Equal(t, tt.want, cliErr.Message)
		})
	}
}

func TestCLIErrorError(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Something failed", (&CLIError{Message: "Something failed", ExitCode: 1}).Error())
}

func TestAPIErrorError(t *testing.T) {
	t.Parallel()
	require.Equal(t, "bitbucket api error (500) for https://api.bitbucket.org/2.0/user: boom", (&APIError{Message: "boom", StatusCode: 500, URL: "https://api.bitbucket.org/2.0/user"}).Error())
}
