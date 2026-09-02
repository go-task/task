package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestLauncherFiltersAsTheUserTypes(t *testing.T) {
	t.Parallel()

	m := testLauncher()
	m, _, request := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	require.Nil(t, request)
	assert.Equal(t, "p", m.filter)
	assert.Equal(t, []int{0, 2}, m.filtered)

	m, _, _ = m.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	assert.Equal(t, "pu", m.filter)
	assert.Equal(t, []int{2}, m.filtered)
	assert.Contains(t, ansi.Strip(m.View().Content), "publish")
	assert.NotContains(t, ansi.Strip(m.View().Content), "build")

	m, _, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	assert.Equal(t, "p", m.filter)
	m, _, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.Empty(t, m.filter)
	assert.Len(t, m.filtered, 3)
}

func TestLauncherUsesSeparateNormalAndTUIActions(t *testing.T) {
	t.Parallel()

	m := testLauncher()
	m, _, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, _, request := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, request)
	assert.Equal(t, "lint", request.name)
	assert.Equal(t, launchInTUI, request.mode)

	_, _, request = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.NotNil(t, request)
	assert.Equal(t, "lint", request.name)
	assert.Equal(t, launchNormally, request.mode)

	_, _, request = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	require.NotNil(t, request)
	assert.Equal(t, launchNormally, request.mode)
}

func TestLauncherEscapeOnlyClearsTheFilter(t *testing.T) {
	t.Parallel()

	m := testLauncher()
	m.applyFilter("build")
	m, cmd, request := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Nil(t, cmd)
	require.Nil(t, request)
	assert.Empty(t, m.filter)

	m, cmd, request = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Nil(t, cmd)
	require.Nil(t, request)
	assert.Empty(t, m.filter)
}

func TestLauncherControlCQuits(t *testing.T) {
	t.Parallel()

	m := testLauncher()
	_, cmd, request := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.Nil(t, request)
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}

func TestLauncherRowsUseNameAndDescriptionColumns(t *testing.T) {
	t.Parallel()

	withoutDescription := launcherRow(launcherItem{name: "lint"}, 40, 10, false)
	withDescription := launcherRow(launcherItem{name: "build", description: "Compile the project"}, 40, 10, true)

	for _, line := range []string{withoutDescription, withDescription} {
		assert.Equal(t, 40, lipgloss.Width(line))
		assert.NotContains(t, ansi.Strip(line), "│")
	}
	assert.Contains(t, ansi.Strip(withDescription), "build       Compile the project")
}

func TestLauncherViewFitsTheTerminalAndScrollsSelection(t *testing.T) {
	t.Parallel()

	m := testLauncher()
	m, _, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 7})
	for range 2 {
		m, _, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	assert.Equal(t, 2, m.selected)
	assert.Greater(t, m.top, 0)
	assert.Equal(t, 60, lipgloss.Width(m.View().Content))
	assert.Equal(t, 7, lipgloss.Height(m.View().Content))
}

func TestAppRunsNormalLauncherSelectionOutsideDashboard(t *testing.T) {
	t.Parallel()

	var normalTask string
	m := newAppModel(
		testLauncher(),
		newTUIModel(func() {}),
		true,
		func() (launcherModel, error) {
			t.Fatal("launcher loader should not run")
			return launcherModel{}, nil
		},
		func([]string) context.CancelFunc {
			t.Fatal("dashboard callback should not run")
			return func() {}
		},
		func(name string) { normalTask = name },
	)

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
	assert.Equal(t, "build", normalTask)
	assert.Equal(t, launcherPage, next.(appModel).page)
}

func TestAppCanReturnToLauncherAfterDashboardExecution(t *testing.T) {
	t.Parallel()

	execution := newTUIModel(func() {})
	execution.canReturnToLauncher = true
	var dashboardTasks []string
	m := newAppModel(
		testLauncher(),
		execution,
		true,
		func() (launcherModel, error) {
			t.Fatal("launcher loader should not run")
			return launcherModel{}, nil
		},
		func(names []string) context.CancelFunc {
			dashboardTasks = append(dashboardTasks, names[0])
			return func() {}
		},
		func(string) { t.Fatal("normal callback should not run") },
	)

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(appModel)
	assert.Equal(t, executionPage, m.page)
	assert.Equal(t, []string{"build"}, dashboardTasks)
	assert.True(t, m.execution.canReturnToLauncher)

	next, _ = m.Update(executionDoneMsg{})
	m = next.(appModel)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(appModel)
	require.NotNil(t, cmd)

	next, _ = m.Update(cmd())
	m = next.(appModel)
	assert.Equal(t, launcherPage, m.page)

	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(appModel)
	assert.Equal(t, executionPage, m.page)
	assert.Equal(t, []string{"build", "build"}, dashboardTasks)
	assert.False(t, m.execution.done)
}

func TestAppLoadsLauncherAfterDirectExecution(t *testing.T) {
	t.Parallel()

	execution := newTUIModel(func() {})
	execution.done = true
	execution.canReturnToLauncher = true
	loaded := false
	m := newAppModel(
		launcherModel{},
		execution,
		false,
		func() (launcherModel, error) {
			loaded = true
			return testLauncher(), nil
		},
		func([]string) context.CancelFunc {
			t.Fatal("dashboard callback should not run")
			return func() {}
		},
		func(string) { t.Fatal("normal callback should not run") },
	)

	next, cmd := m.Update(returnToLauncherMsg{})
	m = next.(appModel)
	assert.True(t, loaded)
	assert.True(t, m.launcherLoaded)
	assert.Equal(t, launcherPage, m.page)
	require.NotNil(t, cmd)
	require.NotEmpty(t, m.launcher.items)
	assert.Equal(t, "build", m.launcher.items[0].name)
}

func TestHelpStylesKeysSeparatelyFromActions(t *testing.T) {
	t.Parallel()

	help := renderControls(80, helpControl{key: "enter", action: "run in TUI"})
	assert.Equal(t, " enter: run in TUI", ansi.Strip(help))
	assert.Contains(t, help, tuiKeyStyle.Render("enter"))
}

func TestLauncherHelpDescribesLaunchActions(t *testing.T) {
	t.Parallel()

	m := testLauncher()
	m.width = 160
	help := ansi.Strip(m.View().Content)
	assert.Contains(t, help, "↑/↓: navigate")
	assert.Contains(t, help, "enter: run in TUI")
	assert.Contains(t, help, "ctrl+r: run normally")
}

func testLauncher() launcherModel {
	return newLauncherModel([]*ast.Task{
		{Task: "build", Desc: "Compile the project"},
		{Task: "lint"},
		{Task: "publish", Desc: "Upload release artifacts"},
	})
}
