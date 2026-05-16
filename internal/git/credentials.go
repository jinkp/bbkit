package git

import (
	"bytes"
	"fmt"
	"net/mail"
	"os/exec"
	"strings"
	"time"
)

type GitCredentialInfo struct {
	Username string
	Password string
}

func GetGitCredential(host string) (*GitCredentialInfo, error) {
	if host == "" {
		host = "bitbucket.org"
	}

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("protocol=https\nhost=%s\n\n", host))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := runWithTimeout(cmd, 5*time.Second); err != nil {
		if stdout.Len() == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("run git credential fill: %w", err)
	}

	username := lookupGitCredentialField(stdout.String(), "username")
	if username == "" {
		return nil, nil
	}

	return &GitCredentialInfo{
		Username: username,
		Password: lookupGitCredentialField(stdout.String(), "password"),
	}, nil
}

func GetGitUserEmail() (string, error) {
	output, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		return "", fmt.Errorf("read git user.email: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func IsEmail(s string) bool {
	address, err := mail.ParseAddress(s)
	return err == nil && address.Address == s
}

func lookupGitCredentialField(output, key string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimSpace(strings.TrimPrefix(line, key+"="))
		}
	}

	return ""
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	timer := time.AfterFunc(timeout, func() {
		_ = cmd.Process.Kill()
	})
	defer timer.Stop()

	return cmd.Run()
}
