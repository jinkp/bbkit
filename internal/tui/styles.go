package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

var (
	colorPrimary = lipgloss.Color("6")
	colorSuccess = lipgloss.Color("2")
	colorError   = lipgloss.Color("1")
	colorDim     = lipgloss.Color("8")
	colorAccent  = lipgloss.Color("5")

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	promptStyle  = lipgloss.NewStyle().Foreground(colorAccent)
	inputStyle   = lipgloss.NewStyle().Foreground(colorPrimary)
	errorStyle   = lipgloss.NewStyle().Foreground(colorError)
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	dimStyle     = lipgloss.NewStyle().Foreground(colorDim)
	spinnerStyle = lipgloss.NewStyle().Foreground(colorPrimary)
)

var spinnerModel = func() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return s
}()
