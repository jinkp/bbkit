// Package opencode handles reading and merging bbkit's MCP entry into opencode.json.
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scope selects which opencode.json file to target.
type Scope string

const (
	// ScopeGlobal targets ~/.config/opencode/opencode.json.
	ScopeGlobal Scope = "global"
	// ScopeLocal targets ./opencode.json in the current working directory.
	ScopeLocal Scope = "local"
)

// bbkitEntry is the MCP server descriptor written into opencode.json.
// OpenCode requires command as an array and an explicit enabled flag.
type bbkitEntry struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
	Enabled bool     `json:"enabled"`
}

// GlobalPath returns the path to the global opencode.json file.
func GlobalPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "opencode", "opencode.json"), nil
}

// LocalPath returns the resolved local opencode.json path.
// Lookup order:
//  1. .opencode/opencode.json  (preferred — OpenCode project convention)
//  2. opencode.json            (fallback for repos that use the root file)
//
// If neither exists, returns .opencode/opencode.json so it gets created there.
func LocalPath() string {
	dotOpencode := filepath.Join(".opencode", "opencode.json")
	if _, err := os.Stat(dotOpencode); err == nil {
		return dotOpencode
	}
	if _, err := os.Stat("opencode.json"); err == nil {
		return "opencode.json"
	}
	// Neither exists → create in .opencode/ (OpenCode convention)
	return dotOpencode
}

// configPath returns the file path for the given scope.
func configPath(scope Scope) (string, error) {
	switch scope {
	case ScopeGlobal:
		return GlobalPath()
	case ScopeLocal:
		return LocalPath(), nil
	default:
		return "", fmt.Errorf("unknown scope %q: must be %q or %q", scope, ScopeGlobal, ScopeLocal)
	}
}

// Load reads path as a JSONC object (JSON with // and /* */ comments) and
// returns it as map[string]json.RawMessage.
// If the file is missing or empty, an empty map is returned (not an error).
// If the file contains malformed JSON, an error is returned.
func Load(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]json.RawMessage), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	if len(data) == 0 {
		return make(map[string]json.RawMessage), nil
	}

	// Strip JSONC comments before parsing — opencode.json files may contain
	// // line comments and /* block comments */ that the standard JSON parser rejects.
	stripped := stripJSONCComments(data)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(stripped, &m); err != nil {
		return nil, fmt.Errorf("parse %s: invalid JSON: %w", path, err)
	}

	if m == nil {
		return make(map[string]json.RawMessage), nil
	}

	return m, nil
}

// stripJSONCComments removes // line comments and /* block comments */ from
// JSON data while preserving content inside strings.
func stripJSONCComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		// Inside a string — copy verbatim until closing quote.
		if data[i] == '"' {
			out = append(out, data[i])
			i++
			for i < len(data) {
				ch := data[i]
				out = append(out, ch)
				i++
				if ch == '\\' && i < len(data) {
					// escaped character — copy next byte and continue
					out = append(out, data[i])
					i++
					continue
				}
				if ch == '"' {
					break
				}
			}
			continue
		}

		// Line comment: // ... \n
		if i+1 < len(data) && data[i] == '/' && data[i+1] == '/' {
			i += 2
			for i < len(data) && data[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment: /* ... */
		if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
			i += 2
			for i+1 < len(data) {
				if data[i] == '*' && data[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		out = append(out, data[i])
		i++
	}
	return out
}

// bbkitEntryJSON is the exact JSON snippet injected into the mcp section.
const bbkitEntryJSON = `{
      "type": "local",
      "command": ["bbk", "mcp"],
      "enabled": true
    }`

// Save merges the bbkit MCP entry into the opencode.json file at the path
// determined by scope. It uses a text-level patch strategy to preserve
// comments, trailing commas, and original formatting.
//
// Strategy:
//  1. File missing or empty → create a fresh minimal JSON file.
//  2. File exists with a "mcp" section → inject "bbkit" entry into it,
//     preserving all surrounding text (comments, formatting, other entries).
//  3. File exists without a "mcp" section → append "mcp" key before the
//     closing brace, preserving all original content.
func Save(scope Scope) error {
	path, err := configPath(scope)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var result []byte
	if len(data) == 0 {
		result = newMinimalConfig()
	} else {
		result, err = patchConfig(data)
		if err != nil {
			return err
		}
	}

	if err := os.WriteFile(path, result, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// newMinimalConfig returns a fresh opencode.json with only the bbkit MCP entry.
func newMinimalConfig() []byte {
	return []byte(`{
  "mcp": {
    "bbkit": ` + bbkitEntryJSON + `
  }
}
`)
}

// patchConfig injects the bbkit MCP entry into existing config bytes.
// It operates at the text level to preserve comments and formatting.
func patchConfig(data []byte) ([]byte, error) {
	text := string(data)

	// Check whether a "bbkit" entry already exists in the stripped version.
	stripped := string(stripJSONCComments(data))
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stripped), &root); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Check if bbkit already present — if so, no-op.
	if mcpRaw, ok := root["mcp"]; ok {
		var mcpMap map[string]json.RawMessage
		if err := json.Unmarshal(mcpRaw, &mcpMap); err == nil {
			if _, exists := mcpMap["bbkit"]; exists {
				return data, nil // already configured
			}
		}
	}

	if _, hasMCP := root["mcp"]; hasMCP {
		// Inject bbkit into the existing "mcp": { ... } block.
		// Find the opening brace of the mcp object in the original text.
		result := injectInteMCPBlock(text)
		if result != "" {
			return []byte(result), nil
		}
	}

	// No "mcp" key found — append it before the final closing brace.
	result := appendMCPSection(text)
	return []byte(result), nil
}

// injectInteMCPBlock finds "mcp": { in the original text and injects the
// bbkit entry as the first item, preserving everything else.
func injectInteMCPBlock(text string) string {
	// Find the "mcp" key followed by its opening brace.
	mcpKey := `"mcp"`
	idx := strings.Index(text, mcpKey)
	if idx < 0 {
		return ""
	}

	// Advance past "mcp" to find the opening '{' of its value.
	after := idx + len(mcpKey)
	braceIdx := strings.Index(text[after:], "{")
	if braceIdx < 0 {
		return ""
	}
	bracePos := after + braceIdx

	// Determine indentation from the line containing "mcp".
	indent := detectIndent(text, idx)

	entry := indent + `  "bbkit": ` + indentBlock(bbkitEntryJSON, indent+"  ") + ","

	// Check if the mcp block is empty (only whitespace between { and })
	afterBrace := text[bracePos+1:]
	stripped := strings.TrimSpace(afterBrace)
	if strings.HasPrefix(stripped, "}") {
		// Empty mcp block — insert without trailing comma on entry
		entryNoComma := indent + `  "bbkit": ` + indentBlock(bbkitEntryJSON, indent+"  ")
		return text[:bracePos+1] + "\n" + entryNoComma + "\n" + indent + text[bracePos+1:]
	}

	return text[:bracePos+1] + "\n" + entry + "\n" + text[bracePos+1:]
}

// appendMCPSection appends a new "mcp" block before the final closing brace.
func appendMCPSection(text string) string {
	lastBrace := strings.LastIndex(text, "}")
	if lastBrace < 0 {
		return text
	}

	mcpBlock := `,
  "mcp": {
    "bbkit": ` + bbkitEntryJSON + `
  }`

	return text[:lastBrace] + mcpBlock + "\n}"
}

// detectIndent returns the leading whitespace of the line containing pos.
func detectIndent(text string, pos int) string {
	lineStart := strings.LastIndex(text[:pos], "\n")
	if lineStart < 0 {
		lineStart = 0
	} else {
		lineStart++
	}
	line := text[lineStart:]
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// indentBlock adds a prefix to every line of block after the first.
func indentBlock(block, prefix string) string {
	lines := strings.Split(block, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
