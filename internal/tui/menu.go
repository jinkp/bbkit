package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jinkp/bbkit/internal/version"
)

type MenuItem struct {
	Label       string
	Description string
	Command     string
}

type MenuResult struct {
	Command   string
	Cancelled bool
}

type MenuModel struct {
	items    []MenuItem
	cursor   int
	result   MenuResult
	quitting bool
}

var menuItems = []MenuItem{
	{Label: "Setup", Description: "Configure bbkit credentials and workspace", Command: "setup"},
	{Label: "Auth Login", Description: "Authenticate with Bitbucket Cloud", Command: "auth login"},
	{Label: "PR List", Description: "List pull requests", Command: "pr list"},
	{Label: "Repo List", Description: "List repositories", Command: "repo list"},
	{Label: "Branch List", Description: "List branches", Command: "branch list"},
	{Label: "Pipeline List", Description: "List recent pipelines", Command: "pipeline list"},
	{Label: "Config", Description: "View current configuration", Command: "config list"},
	{Label: "Setup OpenCode", Description: "Wire bbkit as MCP in OpenCode", Command: "setup opencode"},
	{Label: "Setup Claude Code", Description: "Wire bbkit as MCP in Claude Code", Command: "setup claude"},
}

func NewMenuModel() MenuModel {
	return MenuModel{
		items:  menuItems,
		cursor: 0,
	}
}

func (m MenuModel) Result() MenuResult {
	return m.result
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.result = MenuResult{Cancelled: true}
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			m.result = MenuResult{Command: m.items[m.cursor].Command}
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Brand box
	b.WriteString(renderBrand())
	b.WriteString("\n\n")

	// Menu items
	for i, item := range m.items {
		if i == m.cursor {
			selected := menuSelectedStyle.Render(fmt.Sprintf(" > %s", item.Label))
			desc := menuDimStyle.Render(fmt.Sprintf("  %s", item.Description))
			b.WriteString(selected + "\n")
			b.WriteString(desc + "\n")
		} else {
			line := menuNormalStyle.Render(fmt.Sprintf("   %s", item.Label))
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(menuDimStyle.Render("  ↑/↓ navigate  enter select  q quit"))

	return b.String()
}

func renderBrand() string {
	title := fmt.Sprintf(" bbkit  v%s ", version.Version)
	subtitle := " Modern CLI for Bitbucket Cloud "

	content := lipgloss.JoinVertical(lipgloss.Left, title, subtitle)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(menuAccentColor).
		Padding(0, 1).
		Render(content)

	return box
}

var (
	menuAccentColor    = lipgloss.Color("6")
	menuSelectedColor  = lipgloss.Color("6")
	menuSelectedStyle  = lipgloss.NewStyle().Foreground(menuSelectedColor).Bold(true)
	menuNormalStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	menuDimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
