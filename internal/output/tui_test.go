package output

import (
	"context"
	"errors"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestBuildTUI(t *testing.T) {
	t.Parallel()

	got, err := BuildFor(&ast.Output{Name: "tui"}, &logger.Logger{AssumeTerm: true})
	require.NoError(t, err)
	assert.IsType(t, &TUI{}, got)
}

func TestTUIModelTracksTasksAndOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: "compiling\r\ndone\r"})
	m = updateTUIModel(t, m, started(2, 0, "test"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 1})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2, err: errors.New("failed")})

	require.Len(t, m.tasks, 2)
	assert.Equal(t, taskSucceeded, m.byID[1].state)
	assert.Equal(t, "compiling\ndone\n", m.byID[1].output)
	assert.Equal(t, taskFailed, m.byID[2].state)
	assert.Equal(t, m.width, lipgloss.Width(m.View().Content))
	assert.Equal(t, m.height, lipgloss.Height(m.View().Content))
	assert.LessOrEqual(t, lipgloss.Width(m.View().Content), m.width)
	assert.LessOrEqual(t, lipgloss.Height(m.View().Content), m.height)
	assert.Equal(t, tea.MouseModeCellMotion, m.View().MouseMode)
	left, right := m.renderPanes(newTUILayout(m.width, m.height))
	assert.Equal(t, lipgloss.Height(left), lipgloss.Height(right))

	m.moveSelection(1)
	assert.Equal(t, uint64(2), m.selectedID)
	assert.Contains(t, m.View().Content, "test")
}

func TestTUIModelFitsMinimumTerminalSize(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 40, Height: 8})
	m = updateTUIModel(t, m, started(1, 0, "a-task-with-a-fairly-long-name"))

	view := m.View().Content
	assert.Equal(t, 40, lipgloss.Width(view))
	assert.Equal(t, 8, lipgloss.Height(view))
	assert.LessOrEqual(t, lipgloss.Width(view), 40)
	assert.LessOrEqual(t, lipgloss.Height(view), 8)
	left, right := m.renderPanes(newTUILayout(40, 8))
	assert.Equal(t, lipgloss.Height(left), lipgloss.Height(right))
}

func TestTUIModelKeepsConcurrentInvocationsSeparate(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "worker"))
	m = updateTUIModel(t, m, started(2, 0, "worker"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "worker", data: "first"})
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "second"})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 1, err: errors.New("failed")})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})

	require.Len(t, m.tasks, 2)
	assert.Equal(t, "first", m.byID[1].output)
	assert.Equal(t, taskFailed, m.byID[1].state)
	assert.Equal(t, "second", m.byID[2].output)
	assert.Equal(t, taskSucceeded, m.byID[2].state)
}

func TestTUIModelBuildsRuntimeInvocationTree(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(5, 0, "other-root"))
	m = updateTUIModel(t, m, started(2, 1, "child"))
	m = updateTUIModel(t, m, started(3, 2, "grandchild"))
	m = updateTUIModel(t, m, started(4, 1, "second-child"))

	rows := m.taskRows()
	assert.Equal(t, []string{"root", "child", "grandchild", "second-child", "other-root"}, rowNames(rows))
	assert.Equal(t, []int{0, 1, 2, 1, 0}, rowDepths(rows))
	assert.Equal(t, []string{"", "├─ ", "│  └─ ", "└─ ", ""}, rowPrefixes(rows))
	assert.Contains(t, m.taskList(30, 10), "▌")
}

func TestTUIModelMouseSelectsTasksAndFocusesPanes(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updateTUIModel(t, m, started(1, 0, "first"))
	m = updateTUIModel(t, m, started(2, 0, "second"))

	// The title occupies row 1 inside the border; the second task is row 3.
	m = updateTUIModel(t, m, tea.MouseClickMsg{X: 5, Y: 3, Button: tea.MouseLeft})
	assert.Equal(t, uint64(2), m.selectedID)
	assert.Equal(t, taskPane, m.focus)

	layout := newTUILayout(m.width, m.height)
	m = updateTUIModel(t, m, tea.MouseClickMsg{X: layout.leftOuterWidth + layout.gap + 2, Y: 3, Button: tea.MouseLeft})
	assert.Equal(t, outputPane, m.focus)
}

func TestTUIModelScrollsAndRemembersEachTaskOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 10})
	m = updateTUIModel(t, m, started(1, 0, "first"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "first", data: numberedLines(30)})
	m = updateTUIModel(t, m, started(2, 0, "second"))
	m.focus = outputPane
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})

	firstOffset := m.viewport.YOffset()
	assert.Greater(t, firstOffset, 0)
	assert.False(t, m.byID[1].followOutput)

	m.selectTask(1)
	m.selectTask(0)
	assert.Equal(t, firstOffset, m.viewport.YOffset())

	layout := newTUILayout(m.width, m.height)
	m = updateTUIModel(t, m, tea.MouseWheelMsg{
		X:      layout.leftOuterWidth + layout.gap + 2,
		Y:      4,
		Button: tea.MouseWheelUp,
	})
	assert.Less(t, m.viewport.YOffset(), firstOffset)
}

func TestTUIModelQuitCancelsExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	m := newTUIModel(cancel)
	key := tea.KeyPressMsg{Code: 'q', Text: "q"}
	next, cmd := m.Update(key)
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	assert.IsType(t, tuiModel{}, next)
}

func TestTUIOutputQueueCoalescesWrites(t *testing.T) {
	t.Parallel()

	tui := &TUI{pending: make(map[uint64]pendingOutput)}
	tui.enqueueOutput(7, "build", "one")
	tui.enqueueOutput(7, "build", " two")

	assert.Equal(t, map[uint64]pendingOutput{7: {name: "build", data: "one two"}}, tui.drainOutput())
	assert.False(t, tui.outputQueued)
}

func started(id, parentID uint64, name string) taskStartedMsg {
	return taskStartedMsg{task: TaskInvocation{ID: id, ParentID: parentID, Name: name}}
}

func updateTUIModel(t *testing.T, m tuiModel, msg tea.Msg) tuiModel {
	t.Helper()
	next, _ := m.Update(msg)
	result, ok := next.(tuiModel)
	require.True(t, ok)
	return result
}

func rowNames(rows []tuiTaskRow) []string {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.task.name
	}
	return names
}

func rowDepths(rows []tuiTaskRow) []int {
	depths := make([]int, len(rows))
	for i, row := range rows {
		depths[i] = row.depth
	}
	return depths
}

func rowPrefixes(rows []tuiTaskRow) []string {
	prefixes := make([]string, len(rows))
	for i, row := range rows {
		prefixes[i] = row.treePrefix
	}
	return prefixes
}

func numberedLines(count int) string {
	var output string
	for i := range count {
		output += fmt.Sprintf("line %02d\n", i)
	}
	return output
}
