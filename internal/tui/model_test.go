package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSetupWizardSuccessFlow(t *testing.T) {
	var savedResult SetupResult
	saveFn := func(result SetupResult) error {
		savedResult = result
		return nil
	}

	m := NewModel(&config.Config{}, saveFn)
	require.Equal(t, ScreenEmail, m.screen)

	// Type email and press enter
	m.emailInput.SetValue("user@example.com")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenToken, m.screen)
	require.Equal(t, "user@example.com", m.email)

	// Type token and press enter -> goes to validating
	m.tokenInput.SetValue("mytoken123")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenValidating, m.screen)

	// Simulate successful validation with workspaces
	model, _ = m.Update(validationResultMsg{workspaces: []string{"workspace-a", "workspace-b"}})
	m = model.(Model)
	require.Equal(t, ScreenWorkspace, m.screen)
	require.True(t, m.useWorkspaceList)
	require.Len(t, m.workspaces, 2)

	// Select first workspace and press enter
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenOutput, m.screen)
	require.Equal(t, "workspace-a", m.workspace)

	// Select output format and press enter -> done
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenDone, m.screen)
	require.True(t, m.Done())
	require.False(t, m.Cancelled())
	require.Equal(t, "user@example.com", savedResult.Email)
	require.Equal(t, "mytoken123", savedResult.Token)
	require.Equal(t, "workspace-a", savedResult.Workspace)
	require.Equal(t, "table", savedResult.OutputFormat)
}

func TestSetupWizardCancelWithCtrlC(t *testing.T) {
	m := NewModel(&config.Config{}, nil)
	require.Equal(t, ScreenEmail, m.screen)

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = model.(Model)
	require.True(t, m.Cancelled())
	require.NotNil(t, cmd)
}

func TestSetupWizardValidationFailure(t *testing.T) {
	m := NewModel(&config.Config{}, nil)

	// Navigate to validating screen
	m.emailInput.SetValue("user@example.com")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	m.tokenInput.SetValue("badtoken")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenValidating, m.screen)

	// Simulate failed validation
	model, _ = m.Update(validationResultMsg{err: errors.New("Invalid credentials. Check your email and API token.")})
	m = model.(Model)
	require.Equal(t, ScreenError, m.screen)
	require.True(t, m.Error())
	require.Contains(t, m.ErrorMessage(), "Invalid credentials")
}

func TestSetupWizardSaveFailure(t *testing.T) {
	saveFn := func(result SetupResult) error {
		return errors.New("disk full")
	}

	m := NewModel(&config.Config{}, saveFn)

	// Navigate through to output screen
	m.emailInput.SetValue("user@example.com")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	m.tokenInput.SetValue("mytoken123")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	// Simulate successful validation with no workspace list (fallback to text input)
	model, _ = m.Update(validationResultMsg{workspaces: nil})
	m = model.(Model)
	require.Equal(t, ScreenWorkspace, m.screen)
	require.False(t, m.useWorkspaceList)

	// Type workspace and enter
	m.workspaceInput.SetValue("my-workspace")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenOutput, m.screen)

	// Select output and enter -> save fails -> error screen
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenError, m.screen)
	require.Contains(t, m.ErrorMessage(), "disk full")
}

func TestSetupWizardEmailValidation(t *testing.T) {
	m := NewModel(&config.Config{}, nil)

	// Empty email rejected
	m.emailInput.SetValue("")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenEmail, m.screen)
	require.Equal(t, "Email is required.", m.validationErr)

	// Invalid email rejected
	m.emailInput.SetValue("not-an-email")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenEmail, m.screen)
	require.Equal(t, "Enter a valid email address.", m.validationErr)
}

func TestSetupWizardTokenValidation(t *testing.T) {
	m := NewModel(&config.Config{}, nil)

	// Navigate to token screen
	m.emailInput.SetValue("user@example.com")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	// Empty token rejected
	m.tokenInput.SetValue("")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenToken, m.screen)
	require.Equal(t, "API token is required.", m.validationErr)
}

func TestSetupWizardWorkspaceValidation(t *testing.T) {
	m := NewModel(&config.Config{}, nil)

	// Navigate to workspace screen (text input mode)
	m.emailInput.SetValue("user@example.com")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	m.tokenInput.SetValue("token123")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	model, _ = m.Update(validationResultMsg{workspaces: nil})
	m = model.(Model)
	require.Equal(t, ScreenWorkspace, m.screen)

	// Empty workspace rejected
	m.workspaceInput.SetValue("")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	require.Equal(t, ScreenWorkspace, m.screen)
	require.Equal(t, "Workspace is required.", m.validationErr)
}
