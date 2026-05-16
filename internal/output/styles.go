package output

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func SetWriters(outWriter, errWriter io.Writer) {
	if outWriter != nil {
		stdout = outWriter
	}
	if errWriter != nil {
		stderr = errWriter
	}
}

func ResetWriters() {
	stdout = os.Stdout
	stderr = os.Stderr
}

func IsTTY() bool {
	fd := os.Stdout.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func Success(s string) string {
	return render(successStyle, s)
}

func Error(s string) string {
	return render(errorStyle, s)
}

func Warning(s string) string {
	return render(warningStyle, s)
}

func Info(s string) string {
	return render(infoStyle, s)
}

func PrintSuccess(s string) {
	_, _ = fmt.Fprintln(stdout, Success(s))
}

func PrintError(s string) {
	_, _ = fmt.Fprintln(stderr, Error(s))
}

func PrintWarning(s string) {
	_, _ = fmt.Fprintln(stderr, Warning(s))
}

func PrintInfo(s string) {
	_, _ = fmt.Fprintln(stdout, Info(s))
}

func render(style lipgloss.Style, s string) string {
	if !IsTTY() {
		return s
	}

	return style.Render(s)
}
