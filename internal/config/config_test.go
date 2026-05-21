package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadReturnsEmptyConfigWhenFileDoesNotExist(t *testing.T) {
	setConfigHome(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, &Config{}, cfg)
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	setConfigHome(t)

	want := &Config{
		Workspace:     "workspace",
		Repo:          "repo",
		Username:      "user@example.com",
		DefaultOutput: "json",
	}
	require.NoError(t, Save(want))

	got, err := Load()
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestValidateDefaultOutput(t *testing.T) {
	require.Error(t, Validate(&Config{DefaultOutput: "xml"}))
	require.NoError(t, Validate(&Config{DefaultOutput: "table"}))
	require.NoError(t, Validate(&Config{DefaultOutput: "json"}))
}

func TestGetWorkspaceEnvOverridesConfig(t *testing.T) {
	setConfigHome(t)

	require.NoError(t, Save(&Config{Workspace: "config-workspace"}))

	// Env overrides config
	t.Setenv(envWorkspace, "env-workspace")
	workspace, err := GetWorkspace()
	require.NoError(t, err)
	require.Equal(t, "env-workspace", workspace)

	// Without env, config value is returned
	t.Setenv(envWorkspace, "")
	workspace, err = GetWorkspace()
	require.NoError(t, err)
	require.Equal(t, "config-workspace", workspace)
}

func TestGetUsernameEnvOverridesConfig(t *testing.T) {
	setConfigHome(t)

	require.NoError(t, Save(&Config{Username: "config-user"}))

	// Env overrides config
	t.Setenv(envUsername, "env-user")
	username, err := GetUsername()
	require.NoError(t, err)
	require.Equal(t, "env-user", username)

	// Without env, config value is returned
	t.Setenv(envUsername, "")
	username, err = GetUsername()
	require.NoError(t, err)
	require.Equal(t, "config-user", username)
}

func TestValidateRejectsInvalidDefaultOutput(t *testing.T) {
	err := Validate(&Config{DefaultOutput: "yaml"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid defaultOutput")
}

func TestValidateAcceptsEmptyDefaultOutput(t *testing.T) {
	require.NoError(t, Validate(&Config{DefaultOutput: ""}))
}

func setConfigHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	return filepath.Join(home, appDirName, configFileName)
}
