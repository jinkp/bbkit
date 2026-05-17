package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- Load tests ---

func TestLoadMissingFileReturnsEmptyMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")

	m, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Empty(t, m)
}

func TestLoadEmptyFileReturnsEmptyMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	m, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Empty(t, m)
}

func TestLoadMalformedJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	require.NoError(t, os.WriteFile(path, []byte(`{not valid json}`), 0o644))

	m, err := Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid JSON")
	require.Nil(t, m)
}

func TestLoadValidJSONReturnsMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	content := `{"theme":"dark","model":"claude-3-5-sonnet"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	m, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Contains(t, m, "theme")
	require.Contains(t, m, "model")

	var theme string
	require.NoError(t, json.Unmarshal(m["theme"], &theme))
	require.Equal(t, "dark", theme)
}

// --- Save tests ---

func TestSaveCreatesFileWithCorrectMCPServersEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")

	err := Save(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	require.Contains(t, m, "mcpServers")

	var mcpServersMap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(m["mcpServers"], &mcpServersMap))
	require.Contains(t, mcpServersMap, "bbkit")

	var entry bbkitMCPServer
	require.NoError(t, json.Unmarshal(mcpServersMap["bbkit"], &entry))
	require.Equal(t, "bbk", entry.Command)
	require.Equal(t, []string{"mcp"}, entry.Args)
}

func TestSavePreservesExistingKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")

	// Write a file with existing keys
	existing := `{"theme":"dark","model":"claude-3-5-sonnet"}`
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

	err := Save(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))

	// Existing keys must be preserved
	require.Contains(t, m, "theme")
	require.Contains(t, m, "model")
	// bbkit mcpServers entry must be added
	require.Contains(t, m, "mcpServers")
}

func TestSaveMergesBBKitIntoExistingMCPServersSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")

	// Write a file with an existing mcpServers section containing another entry
	existing := `{"mcpServers":{"other-tool":{"command":"other","args":[]}}}`
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

	err := Save(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	require.Contains(t, m, "mcpServers")

	var mcpServersMap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(m["mcpServers"], &mcpServersMap))

	// Both the existing entry and the new bbkit entry must be present
	require.Contains(t, mcpServersMap, "other-tool", "pre-existing mcpServers entry must be preserved")
	require.Contains(t, mcpServersMap, "bbkit", "bbkit entry must be added")
}

func TestSaveCreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, ".claude", "settings.json")

	err := Save(nested)
	require.NoError(t, err)

	// Verify the file was created in the nested directory
	_, err = os.Stat(nested)
	require.NoError(t, err, "file should have been created in nested directory")

	data, err := os.ReadFile(nested)
	require.NoError(t, err)

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	require.Contains(t, m, "mcpServers")
}
