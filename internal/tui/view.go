package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	switch m.screen {
	case ScreenEmail:
		return renderScreen(
			"bbkit setup",
			"Step 1 of 4 — Enter your Atlassian email.",
			m.emailInput.View(),
			m.validationErr,
			"Enter or Tab to continue • Ctrl+C to cancel",
		)
	case ScreenToken:
		return renderScreen(
			"bbkit setup",
			"Step 2 of 4 — Paste your Bitbucket API token.",
			m.tokenInput.View(),
			m.validationErr,
			"Enter or Tab to validate • Ctrl+C to cancel",
		)
	case ScreenValidating:
		body := fmt.Sprintf("%s Verifying credentials and fetching workspaces...", m.spinner.View())
		return renderScreen(
			"bbkit setup",
			"Step 3 of 4 — Validating with Bitbucket Cloud.",
			spinnerStyle.Render(body),
			"",
			"Ctrl+C to cancel",
		)
	case ScreenWorkspace:
		body := m.workspaceInput.View()
		hint := "Enter or Tab to continue • Ctrl+C to cancel"
		if m.useWorkspaceList {
			body = renderChoices(m.workspaces, m.workspaceCursor, nil)
			hint = "↑/↓ to select • Enter or Tab to continue • Ctrl+C to cancel"
		}
		return renderScreen(
			"bbkit setup",
			"Step 3 of 4 — Choose your default workspace.",
			body,
			m.validationErr,
			hint,
		)
	case ScreenOutput:
		options := []string{"Table", "JSON"}
		hints := []string{"human-friendly (default)", "machine-readable, piping"}
		return renderScreen(
			"bbkit setup",
			"Step 4 of 4 — Choose the default output format.",
			renderChoices(options, m.outputIdx, hints),
			m.validationErr,
			"↑/↓ to select • Enter or Tab to save • Ctrl+C to cancel",
		)
	case ScreenDone:
		result := m.Result()
		summary := []string{
			successStyle.Render("Configuration saved"),
			fmt.Sprintf("Email: %s", result.Email),
			fmt.Sprintf("Workspace: %s", result.Workspace),
			fmt.Sprintf("Output: %s", result.OutputFormat),
			"Credentials stored in OS keychain",
		}
		return renderScreen(
			"bbkit setup",
			"Ready to go.",
			strings.Join(summary, "\n"),
			"",
			"Press Enter to finish",
		)
	case ScreenError:
		return renderScreen(
			"bbkit setup",
			"Setup failed.",
			errorStyle.Render(m.validationErr),
			"",
			"Run `bbk setup` to try again • Press Enter to exit",
		)
	default:
		return ""
	}
}

func renderScreen(title, subtitle, body, errText, hint string) string {
	parts := []string{
		titleStyle.Render(title),
		dimStyle.Render(subtitle),
		"",
		body,
	}

	if strings.TrimSpace(errText) != "" {
		parts = append(parts, "", errorStyle.Render(errText))
	}

	if strings.TrimSpace(hint) != "" {
		parts = append(parts, "", dimStyle.Render(hint))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(parts, "\n"))
}

func renderChoices(options []string, cursor int, hints []string) string {
	lines := make([]string, 0, len(options))
	for i, option := range options {
		prefix := "  "
		style := lipgloss.NewStyle()
		if i == cursor {
			prefix = promptStyle.Render("› ")
			style = inputStyle
		}

		line := prefix + style.Render(option)
		if i < len(hints) && strings.TrimSpace(hints[i]) != "" {
			line += " " + dimStyle.Render("— "+hints[i])
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
