package tui

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func Run() error {
	m := newMainModel()
	p := tea.NewProgram(m)

	_, err := p.Run()
	return err
}
