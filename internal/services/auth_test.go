package services

import (
	"testing"

	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/secrets"
	"github.com/stretchr/testify/require"
)

func TestStatusReturnsEnvSourceWhenEnvVarsSet(t *testing.T) {
	reset := stubAuthDeps(t)
	defer reset()

	secretsGetWithSource = func() (*secrets.Credentials, string, error) {
		return &secrets.Credentials{Username: "env-user", APIToken: "env-token"}, "env", nil
	}

	status, err := Status()
	require.NoError(t, err)
	require.True(t, status.Authenticated)
	require.Equal(t, "env", status.Source)
}

func TestStatusReturnsKeyringSource(t *testing.T) {
	reset := stubAuthDeps(t)
	defer reset()

	secretsGetWithSource = func() (*secrets.Credentials, string, error) {
		return &secrets.Credentials{Username: "testuser", APIToken: "token123"}, "keyring", nil
	}

	status, err := Status()
	require.NoError(t, err)
	require.True(t, status.Authenticated)
	require.Equal(t, "keyring", status.Source)
}

func TestStatusReturnsNotAuthenticatedWhenNoCreds(t *testing.T) {
	reset := stubAuthDeps(t)
	defer reset()

	secretsGetWithSource = func() (*secrets.Credentials, string, error) { return nil, "", nil }

	status, err := Status()
	require.NoError(t, err)
	require.False(t, status.Authenticated)
}

func TestLogoutClearsCredentials(t *testing.T) {
	reset := stubAuthDeps(t)
	defer reset()

	called := false
	secretsClear = func() error {
		called = true
		return nil
	}

	require.NoError(t, Logout())
	require.True(t, called)
}

func TestGetCredentialsOrFailReturnsCLIErrorWhenNoCreds(t *testing.T) {
	reset := stubAuthDeps(t)
	defer reset()

	secretsGet = func() (*secrets.Credentials, error) { return nil, nil }

	creds, err := GetCredentialsOrFail()
	require.Nil(t, creds)
	var cliErr *bitbucket.CLIError
	require.ErrorAs(t, err, &cliErr)
	require.Contains(t, cliErr.Message, "Not authenticated")
}

func stubAuthDeps(t *testing.T) func() {
	t.Helper()
	originalGetWithSource := secretsGetWithSource
	originalClear := secretsClear
	originalGet := secretsGet
	originalSave := secretsSave
	originalConfigSave := configSave
	return func() {
		secretsGetWithSource = originalGetWithSource
		secretsClear = originalClear
		secretsGet = originalGet
		secretsSave = originalSave
		configSave = originalConfigSave
	}
}
