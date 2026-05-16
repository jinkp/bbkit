package services

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/jinkp/bbkit/internal/bitbucket"
)

func CheckoutSourceBranch(ctx context.Context, branchName string) error {
	if !isSafeBranchName(branchName) {
		return &bitbucket.CLIError{Message: "Unsafe source branch name: '" + branchName + "'.", ExitCode: 1}
	}

	fetch := exec.CommandContext(ctx, "git", "fetch", "origin")
	fetch.Stdout = os.Stdout
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		return &bitbucket.CLIError{Message: "Could not check out source branch '" + branchName + "'.", ExitCode: 1}
	}

	checkout := exec.CommandContext(ctx, "git", "checkout", branchName)
	checkout.Stdout = os.Stdout
	checkout.Stderr = os.Stderr
	if err := checkout.Run(); err != nil {
		return &bitbucket.CLIError{Message: "Could not check out source branch '" + branchName + "'.", ExitCode: 1}
	}

	return nil
}

func isSafeBranchName(branchName string) bool {
	trimmed := strings.TrimSpace(branchName)
	if trimmed == "" || trimmed != branchName || strings.HasPrefix(trimmed, "-") {
		return false
	}

	for _, char := range branchName {
		if char <= 0x1f || char == 0x7f || char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			return false
		}
	}

	return true
}
