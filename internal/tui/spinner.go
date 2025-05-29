package tui

import (
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(charmtone.Violet.Hex()))

func newSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(spinnerStyle),
	)
}
