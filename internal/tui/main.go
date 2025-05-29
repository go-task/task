package tui

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type page int

const (
	pageList = iota + 1
)

type mainModel struct {
	page       page
	listPage   listModel
	isQuitting bool
}

func newMainModel() mainModel {
	return mainModel{
		page:     pageList,
		listPage: newListModel(),
	}
}

func (m mainModel) Init() tea.Cmd {
	return m.listPage.Init()
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			m.isQuitting = true
			return m, tea.Quit
		}
	}

	switch m.page {
	case pageList:
		model, cmd := m.listPage.Update(msg)
		m.listPage = model.(listModel)
		return m, cmd
	}

	return m, nil
}

func (m mainModel) View() string {
	switch {
	case m.isQuitting:
		return "Quitting..."
	case m.page == pageList:
		return m.listPage.View()
	default:
		return "..."
	}
}
