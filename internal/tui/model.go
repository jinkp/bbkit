package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/bbkit/internal/config"
	"github.com/jinkp/bbkit/internal/git"
)

type Screen int

const (
	ScreenEmail Screen = iota
	ScreenToken
	ScreenValidating
	ScreenWorkspace
	ScreenOutput
	ScreenDone
	ScreenError
)

type SetupResult struct {
	Email        string
	Token        string
	Workspace    string
	OutputFormat string
}

type SaveSetupFunc func(SetupResult) error

type Model struct {
	screen           Screen
	width, height    int
	emailInput       textinput.Model
	tokenInput       textinput.Model
	workspaceInput   textinput.Model
	workspaces       []string
	workspaceCursor  int
	useWorkspaceList bool
	outputIdx        int
	spinner          spinner.Model
	validationErr    string
	doneMsg          string
	email            string
	token            string
	workspace        string
	outputFormat     string
	cfg              *config.Config
	saveFn           SaveSetupFunc
	cancelled        bool
	cancelMsg        string
	defaultWorkspace string
}

type validationResultMsg struct {
	err        error
	workspaces []string
}

func NewModel(cfg *config.Config, saveFn SaveSetupFunc) Model {
	if cfg == nil {
		cfg = &config.Config{}
	}

	detectedEmail := detectEmailHint()
	detectedWorkspace := detectWorkspaceHint(cfg)

	emailInput := newTextInput("you@company.com", detectedEmail, false)
	emailInput.Focus()

	tokenInput := newTextInput("Paste your Bitbucket API token", "", true)

	workspaceInput := newTextInput("my-company", detectedWorkspace, false)

	outputIdx := 0
	if strings.EqualFold(strings.TrimSpace(cfg.DefaultOutput), "json") {
		outputIdx = 1
	}

	return Model{
		screen:           ScreenEmail,
		emailInput:       emailInput,
		tokenInput:       tokenInput,
		workspaceInput:   workspaceInput,
		outputIdx:        outputIdx,
		outputFormat:     outputFormatValue(outputIdx),
		spinner:          spinnerModel,
		cfg:              cfg,
		saveFn:           saveFn,
		cancelMsg:        "Setup cancelled.",
		defaultWorkspace: detectedWorkspace,
	}
}

func (m Model) Cancelled() bool {
	return m.cancelled
}

func (m Model) Done() bool {
	return m.screen == ScreenDone
}

func (m Model) Error() bool {
	return m.screen == ScreenError
}

func (m Model) CancelMessage() string {
	return m.cancelMsg
}

func (m Model) DoneMessage() string {
	if strings.TrimSpace(m.doneMsg) != "" {
		return m.doneMsg
	}
	return "Setup complete. Run `bbk repo list` to get started."
}

func (m Model) ErrorMessage() string {
	if strings.TrimSpace(m.validationErr) == "" {
		return "Setup failed. Run `bbk setup` to try again."
	}
	return m.validationErr + "\nRun `bbk setup` to try again."
}

func (m Model) Result() SetupResult {
	return SetupResult{
		Email:        strings.TrimSpace(firstNonEmpty(m.email, m.emailInput.Value())),
		Token:        strings.TrimSpace(firstNonEmpty(m.token, m.tokenInput.Value())),
		Workspace:    strings.TrimSpace(firstNonEmpty(m.workspace, m.selectedWorkspace(), m.workspaceInput.Value())),
		OutputFormat: outputFormatValue(m.outputIdx),
	}
}

func newTextInput(placeholder, value string, masked bool) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.Prompt = "> "
	ti.CharLimit = 200
	ti.Width = 48
	ti.PromptStyle = promptStyle
	ti.TextStyle = inputStyle
	ti.PlaceholderStyle = dimStyle
	ti.Cursor.Style = inputStyle
	if masked {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '*'
	}
	return ti
}

func detectEmailHint() string {
	gitCred, err := git.GetGitCredential("bitbucket.org")
	if err == nil && gitCred != nil && git.IsEmail(gitCred.Username) {
		return strings.TrimSpace(gitCred.Username)
	}

	gitEmail, err := git.GetGitUserEmail()
	if err == nil && git.IsEmail(gitEmail) {
		return strings.TrimSpace(gitEmail)
	}

	return ""
}

func detectWorkspaceHint(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Workspace) != "" {
		return strings.TrimSpace(cfg.Workspace)
	}

	remote, err := git.InferFromGitRemote()
	if err == nil && remote != nil {
		return strings.TrimSpace(remote.Workspace)
	}

	return ""
}

func (m *Model) focusWorkspaceInput() {
	m.emailInput.Blur()
	m.tokenInput.Blur()
	m.workspaceInput.Focus()
}

func (m Model) selectedWorkspace() string {
	if !m.useWorkspaceList || len(m.workspaces) == 0 {
		return ""
	}
	if m.workspaceCursor < 0 || m.workspaceCursor >= len(m.workspaces) {
		return ""
	}
	return m.workspaces[m.workspaceCursor]
}

func outputFormatValue(idx int) string {
	if idx == 1 {
		return "json"
	}
	return "table"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}
