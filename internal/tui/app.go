package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

type appPage uint8

const (
	launcherPage appPage = iota
	executionPage
)

type appModel struct {
	page      appPage
	launcher  launcherModel
	execution tuiModel
	startTUI  func([]string) context.CancelFunc
	runNormal func(string)
	width     int
	height    int
}

func newAppModel(
	launcher launcherModel,
	execution tuiModel,
	showLauncher bool,
	startTUI func([]string) context.CancelFunc,
	runNormal func(string),
) appModel {
	page := executionPage
	if showLauncher {
		page = launcherPage
	}
	return appModel{
		page:      page,
		launcher:  launcher,
		execution: execution,
		startTUI:  startTUI,
		runNormal: runNormal,
	}
}

func (m appModel) Init() tea.Cmd {
	if m.page == launcherPage {
		return m.launcher.Init()
	}
	return m.execution.Init()
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
	}

	if m.page == launcherPage {
		if _, ok := msg.(interruptRequestedMsg); ok {
			return m, tea.Quit
		}
		launcher, cmd, request := m.launcher.Update(msg)
		m.launcher = launcher
		if request == nil {
			return m, cmd
		}
		if request.mode == launchNormally {
			m.runNormal(request.name)
			return m, tea.Quit
		}

		m.page = executionPage
		statusLabels, taskNavigator := m.execution.statusLabels, m.execution.taskNavigator
		canReturnToLauncher := m.execution.canReturnToLauncher
		cancel := m.startTUI([]string{request.name})
		m.execution = newTUIModel(cancel)
		m.execution.statusLabels = statusLabels
		m.execution.taskNavigator = taskNavigator
		m.execution.canReturnToLauncher = canReturnToLauncher
		execution, resizeCmd := m.execution.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.execution = execution.(tuiModel)
		return m, tea.Batch(cmd, resizeCmd, tea.ClearScreen)
	}
	if _, ok := msg.(returnToLauncherMsg); ok {
		m.page = launcherPage
		launcher, _, _ := m.launcher.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.launcher = launcher
		return m, tea.ClearScreen
	}

	execution, cmd := m.execution.Update(msg)
	m.execution = execution.(tuiModel)
	return m, cmd
}

func (m appModel) View() tea.View {
	if m.page == launcherPage {
		return m.launcher.View()
	}
	return m.execution.View()
}
