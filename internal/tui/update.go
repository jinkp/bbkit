package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/bbkit/internal/bitbucket"
	"github.com/jinkp/bbkit/internal/git"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}
	}

	switch m.screen {
	case ScreenEmail:
		return m.updateEmail(msg)
	case ScreenToken:
		return m.updateToken(msg)
	case ScreenValidating:
		return m.updateValidating(msg)
	case ScreenWorkspace:
		return m.updateWorkspace(msg)
	case ScreenOutput:
		return m.updateOutput(msg)
	case ScreenDone:
		return m.updateDone(msg)
	case ScreenError:
		return m.updateError(msg)
	default:
		return m, nil
	}
}

func (m Model) updateEmail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "enter":
			email := strings.TrimSpace(m.emailInput.Value())
			if email == "" {
				m.validationErr = "Email is required."
				return m, nil
			}
			if !git.IsEmail(email) {
				m.validationErr = "Enter a valid email address."
				return m, nil
			}

			m.email = email
			m.validationErr = ""
			m.emailInput.Blur()
			m.tokenInput.Focus()
			m.screen = ScreenToken
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.emailInput, cmd = m.emailInput.Update(msg)
	return m, cmd
}

func (m Model) updateToken(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "enter":
			token := strings.TrimSpace(m.tokenInput.Value())
			if token == "" {
				m.validationErr = "API token is required."
				return m, nil
			}

			m.token = token
			m.validationErr = ""
			m.tokenInput.Blur()
			m.screen = ScreenValidating
			return m, validateCredentialsCmd(m.email, m.token)
		}
	}

	var cmd tea.Cmd
	m.tokenInput, cmd = m.tokenInput.Update(msg)
	return m, cmd
}

func (m Model) updateValidating(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case validationResultMsg:
		if msg.err != nil {
			m.validationErr = msg.err.Error()
			m.screen = ScreenError
			return m, nil
		}

		m.validationErr = ""
		m.workspaces = msg.workspaces
		m.useWorkspaceList = len(msg.workspaces) > 0
		m.workspace = ""

		if m.useWorkspaceList {
			m.workspaceCursor = 0
			defaultWorkspace := firstNonEmpty(m.defaultWorkspace, m.cfg.Workspace)
			for i, workspace := range m.workspaces {
				if workspace == defaultWorkspace {
					m.workspaceCursor = i
					break
				}
			}
			m.workspaceInput.Blur()
		} else {
			m.focusWorkspaceInput()
		}

		m.screen = ScreenWorkspace
		return m, nil
	case tea.KeyMsg:
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) updateWorkspace(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.useWorkspaceList {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.workspaceCursor > 0 {
					m.workspaceCursor--
				}
				return m, nil
			case "down", "j":
				if m.workspaceCursor < len(m.workspaces)-1 {
					m.workspaceCursor++
				}
				return m, nil
			case "tab", "enter":
				selected := m.selectedWorkspace()
				if selected == "" {
					m.validationErr = "Workspace is required."
					return m, nil
				}
				m.workspace = selected
				m.validationErr = ""
				m.screen = ScreenOutput
				return m, nil
			}
		}

		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "enter":
			workspace := strings.TrimSpace(m.workspaceInput.Value())
			if workspace == "" {
				m.validationErr = "Workspace is required."
				return m, nil
			}
			m.workspace = workspace
			m.validationErr = ""
			m.workspaceInput.Blur()
			m.screen = ScreenOutput
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.workspaceInput, cmd = m.workspaceInput.Update(msg)
	return m, cmd
}

func (m Model) updateOutput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "left", "k", "h":
			if m.outputIdx > 0 {
				m.outputIdx--
			}
			return m, nil
		case "down", "right", "j", "l":
			if m.outputIdx < 1 {
				m.outputIdx++
			}
			return m, nil
		case "tab", "enter":
			m.outputFormat = outputFormatValue(m.outputIdx)
			if m.saveFn != nil {
				if err := m.saveFn(m.Result()); err != nil {
					m.validationErr = err.Error()
					m.screen = ScreenError
					return m, nil
				}
			}
			m.doneMsg = "Setup complete. Run `bbk repo list` to get started."
			m.screen = ScreenDone
			return m, nil
		}
	}

	return m, nil
}

func (m Model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "enter", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "enter", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func validateCredentialsCmd(email, token string) tea.Cmd {
	return func() tea.Msg {
		client := bitbucket.NewClient(email, token)
		ctx := context.Background()

		var user map[string]any
		if err := client.Get(ctx, "/user", &user); err != nil {
			return validationResultMsg{err: mapValidationError(err)}
		}

		workspaces, err := bitbucket.Paginate[bitbucket.Workspace](ctx, client, "/workspaces?pagelen=50")
		if err != nil {
			return validationResultMsg{workspaces: nil}
		}

		slugs := make([]string, 0, len(workspaces))
		seen := make(map[string]struct{}, len(workspaces))
		for _, workspace := range workspaces {
			slug := strings.TrimSpace(workspace.Slug)
			if slug == "" {
				continue
			}
			if _, ok := seen[slug]; ok {
				continue
			}
			seen[slug] = struct{}{}
			slugs = append(slugs, slug)
		}

		return validationResultMsg{workspaces: slugs}
	}
}

func mapValidationError(err error) error {
	var cliErr *bitbucket.CLIError
	if errors.As(err, &cliErr) {
		if cliErr.ExitCode == 1 && (strings.Contains(cliErr.Message, "Authentication failed") || strings.Contains(cliErr.Message, "Permission denied")) {
			return fmt.Errorf("Invalid credentials. Check your email and API token.")
		}
		return errors.New(cliErr.Message)
	}
	return err
}
