package services

import (
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jinkp/bbkit/internal/bitbucket"
)

func OpenURL(rawURL string) error {
	safeURL, err := resolveSafeURL(rawURL)
	if err != nil {
		return err
	}

	file, args := openCommand(safeURL)
	if err := exec.Command(file, args...).Start(); err != nil {
		return &bitbucket.CLIError{Message: "Could not open URL in the default browser.", ExitCode: 1}
	}

	return nil
}

func openCommand(target string) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		return "cmd", []string{"/c", "start", "", target}
	case "darwin":
		return "open", []string{target}
	default:
		return "xdg-open", []string{target}
	}
}

func resolveSafeURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed != value {
		return "", &bitbucket.CLIError{Message: "Pull request URL is invalid or unsafe.", ExitCode: 1}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", &bitbucket.CLIError{Message: "Pull request URL is invalid or unsafe.", ExitCode: 1}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", &bitbucket.CLIError{Message: "Pull request URL is invalid or unsafe.", ExitCode: 1}
	}

	return parsed.String(), nil
}
