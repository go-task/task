package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type listModel struct {
	spinner       spinner.Model
	list          list.Model
	selectedIndex int
	isLoading     bool
}

func newListModel() listModel {
	return listModel{
		spinner:   newSpinner(),
		list:      list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		isLoading: true,
	}
}

func (m listModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	switch {
	case m.isLoading:
		return fmt.Sprintf("%s %s", m.spinner.View(), "Loading tasks...")
	default:
		return "todo!"
	}
}
