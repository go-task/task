package tui

import (
	"strings"
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
	assert.Equal(t, launchNormally, request.mode)

	_, _, request = m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	require.NotNil(t, request)
	assert.Equal(t, "lint", request.name)
	assert.Equal(t, launchInTUI, request.mode)
}

func TestLauncherCardsOnlyReserveDescriptionRowsWhenNeeded(t *testing.T) {
	t.Parallel()

	withoutDescription := launcherCard(launcherItem{name: "lint"}, 30, false)
	withDescription := launcherCard(launcherItem{name: "build", description: "Compile the project"}, 30, true)

	assert.Len(t, withoutDescription, 3)
	assert.Len(t, withDescription, 4)
	for _, line := range append(withoutDescription, withDescription...) {
		assert.Equal(t, 30, lipgloss.Width(line))
	}
	assert.Contains(t, ansi.Strip(strings.Join(withDescription, "\n")), "Compile the project")
}

func TestLauncherViewFitsTheTerminalAndScrollsSelection(t *testing.T) {
	t.Parallel()

	m := testLauncher()
	m, _, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	for range 2 {
		m, _, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	assert.Equal(t, 2, m.selected)
	assert.Greater(t, m.top, 0)
	assert.Equal(t, 60, lipgloss.Width(m.View().Content))
	assert.Equal(t, 12, lipgloss.Height(m.View().Content))
}

func TestAppRunsNormalLauncherSelectionOutsideDashboard(t *testing.T) {
	t.Parallel()

	var normalTask string
	m := newAppModel(
		testLauncher(),
		newTUIModel(func() {}),
		true,
		func([]string) { t.Fatal("dashboard callback should not run") },
		func(name string) { normalTask = name },
	)

	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
	assert.Equal(t, "build", normalTask)
	assert.Equal(t, launcherPage, next.(appModel).page)
}

func testLauncher() launcherModel {
	return newLauncherModel([]*ast.Task{
		{Task: "build", Desc: "Compile the project"},
		{Task: "lint"},
		{Task: "publish", Desc: "Upload release artifacts"},
	})
}
