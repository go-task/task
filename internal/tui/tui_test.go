package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/logger"
)

func TestNew(t *testing.T) {
	t.Parallel()

	got, err := New(&logger.Logger{AssumeTerm: true}, Options{})
	require.NoError(t, err)
	assert.Equal(t, taskNavigatorTree, got.taskNavigator)

	got, err = New(&logger.Logger{AssumeTerm: true}, Options{Status: "labels", TaskNavigator: "list"})
	require.NoError(t, err)
	assert.True(t, got.statusLabels)
	assert.Equal(t, taskNavigatorList, got.taskNavigator)

	_, err = New(&logger.Logger{AssumeTerm: true}, Options{Status: "unknown"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `expected "icons" or "labels"`)

	_, err = New(&logger.Logger{AssumeTerm: true}, Options{TaskNavigator: "unknown"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `expected "list" or "tree"`)
}

func TestTUIModelTracksTasksAndOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "build", data: "compiling\r\ndone\r"})
	m = updateTUIModel(t, m, started(3, 1, "test"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 3, err: errors.New("failed")})

	require.Len(t, m.tasks, 3)
	assert.Equal(t, taskSucceeded, m.byID[2].state)
	// The trailing carriage return returns the cursor to the start of "done"
	// without erasing it, so the line stays visible until something redraws it.
	assert.Equal(t, "compiling\ndone", m.byID[2].output)
	assert.Equal(t, taskFailed, m.byID[3].state)
	assert.Equal(t, m.width, lipgloss.Width(m.View().Content))
	assert.Equal(t, m.height, lipgloss.Height(m.View().Content))
	assert.LessOrEqual(t, lipgloss.Width(m.View().Content), m.width)
	assert.LessOrEqual(t, lipgloss.Height(m.View().Content), m.height)
	assert.Equal(t, tea.MouseModeCellMotion, m.View().MouseMode)
	left, right := m.renderPanes(newTUILayout(m.width, m.height))
	assert.Equal(t, lipgloss.Height(left), lipgloss.Height(right))

	m.moveSelection(1)
	assert.Equal(t, uint64(3), m.selectedID)
	assert.Contains(t, m.View().Content, "test")
}

func TestTUIModelDistinguishesCanceledTasksAndShowsStatusWords(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m.statusLabels = true
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, scheduled(2, 1, "pending-task"))
	m = updateTUIModel(t, m, started(3, 1, "running-task"))
	m = updateTUIModel(t, m, started(4, 1, "successful-task"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 4})
	m = updateTUIModel(t, m, started(5, 1, "failed-task"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 5, err: errors.New("failed")})
	m = updateTUIModel(t, m, started(6, 1, "canceled-task"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 6, err: fmt.Errorf("wrapped: %w", context.Canceled)})

	assert.Equal(t, taskCanceled, m.byID[6].state)
	assert.Equal(t, "failed\n", m.byID[5].output)
	assert.Equal(t, "■", taskIconText(taskCanceled))
	list := m.taskList(50, 20)
	for _, status := range []string{"pending", "running", "success", "failed", "canceled"} {
		assert.Contains(t, list, status)
	}
	name, status := taskNameStatus("build", taskRunning, 20, true)
	assert.Equal(t, "build running", name+" "+status)
	name, status = taskNameStatus("fail-fast-success-1s", taskSucceeded, 13, true)
	assert.Equal(t, "fa…1s success", name+" "+status)
}

func TestTUIStatusLabelsAreOptionalAndDisabledByDefault(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))

	icons := ansi.Strip(m.taskList(30, 10))
	assert.Contains(t, icons, "└─ ● worker")
	assert.NotContains(t, icons, "running")
	m.statusLabels = true
	labels := ansi.Strip(m.taskList(30, 10))
	assert.Contains(t, labels, "└─ worker running")
	assert.NotContains(t, labels, "●")
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

func TestTUILayoutGivesWideTerminalsMoreTaskSpace(t *testing.T) {
	t.Parallel()

	compact := newTUILayout(80, 24)
	wide := newTUILayout(240, 24)

	assert.Zero(t, compact.gap)
	assert.Greater(t, wide.leftOuterWidth, compact.leftOuterWidth)
	assert.Equal(t, 72, wide.leftOuterWidth)
	assert.Equal(t, compact.leftOuterWidth-tuiPanelStyle.GetHorizontalFrameSize(), compact.leftInnerWidth)
	assert.Equal(t, compact.rightOuterWidth-tuiPanelStyle.GetHorizontalFrameSize(), compact.rightInnerWidth)
	assert.Equal(t, compact.bodyHeight-tuiPanelStyle.GetVerticalFrameSize(), compact.innerHeight)
}

func TestTUITextTruncationUsesTerminalCellWidth(t *testing.T) {
	t.Parallel()

	assert.LessOrEqual(t, ansi.StringWidth(truncateText("界界界", 4)), 4)
	middle := truncateMiddle("build-界界-target", 10)
	assert.LessOrEqual(t, ansi.StringWidth(middle), 10)
	assert.Contains(t, middle, "…")
	assert.True(t, strings.HasSuffix(middle, "arget"), middle)
}

func TestTUIModelKeepsRepeatedTaskCallsSeparate(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, started(3, 1, "worker"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "first"})
	m = updateTUIModel(t, m, taskOutputMsg{id: 3, name: "worker", data: " second"})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2, err: errors.New("failed")})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 3})

	require.Len(t, m.tasks, 3)
	assert.NotSame(t, m.byID[2], m.byID[3])
	assert.Equal(t, "first\nfailed\n", m.byID[2].output)
	assert.Equal(t, " second", m.byID[3].output)
	assert.Equal(t, taskFailed, m.byID[2].state)
	assert.Equal(t, taskSucceeded, m.byID[3].state)
	assert.Equal(t, []string{"root", "worker", "worker"}, rowNames(m.taskRows()))
	assert.Contains(t, m.taskList(30, 10), "#1 worker")
	assert.Contains(t, m.taskList(30, 10), "#2 worker")

	m.selectTask(2)
	assert.Equal(t, " second", m.selectedTask().output)
	assert.Contains(t, m.viewport.View(), " second")
	assert.Contains(t, m.outputPanel(30), "#2 worker")
}

func TestTUIModelSharesJoinedExecutionStatusAndOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m.taskNavigator = taskNavigatorTree
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "shared output"})
	m = updateTUIModel(t, m, scheduled(3, 1, "worker"))
	require.Len(t, m.tasks, 3)
	assert.Contains(t, m.taskList(30, 10), "#1 worker")
	assert.Contains(t, m.taskList(30, 10), "#2 worker")

	m.selectTask(2)
	m = updateTUIModel(t, m, taskJoinedMsg{id: 3, ownerID: 2})

	require.Len(t, m.tasks, 3)
	assert.Equal(t, uint64(2), m.byID[3].ownerID)
	assert.True(t, m.byID[2].shared)
	assert.True(t, m.byID[3].shared)
	assert.Equal(t, []uint64{1, 2, 3}, rowIDs(m.taskRows()))
	assert.Equal(t, "#1 worker", m.taskName(m.byID[2]))
	assert.Equal(t, "#2 worker", m.taskName(m.byID[3]))
	assert.Equal(t, uint64(3), m.selectedID)
	assert.Equal(t, uint64(2), m.selectedTask().id)
	assert.Equal(t, "shared output", m.selectedTask().output)
	assert.Contains(t, m.outputPanel(30), "#2 worker")
	assert.Equal(t, taskRunning, m.taskState(m.byID[3]))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: " continued"})
	assert.Contains(t, m.viewport.View(), "shared output continued")
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	assert.Equal(t, taskSucceeded, m.taskState(m.byID[3]))
	assert.Equal(t, []string{"root", "worker", "worker"}, rowNames(m.taskRows()))
	assert.Equal(t, 2, strings.Count(ansi.Strip(m.taskList(30, 10)), "↳"))
}

func TestTUIModelShowsSharedExecutionInEachTreeLocation(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m.taskNavigator = taskNavigatorTree
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "parent-a"))
	m = updateTUIModel(t, m, started(3, 1, "parent-b"))
	m = updateTUIModel(t, m, startedUnder(4, 2, 1, "shared"))
	m = updateTUIModel(t, m, scheduledUnder(5, 3, 1, "shared"))
	m = updateTUIModel(t, m, taskJoinedMsg{id: 5, ownerID: 4})

	assert.Equal(t, []string{"root", "parent-a", "shared", "parent-b", "shared"}, rowNames(m.taskRows()))
	list := ansi.Strip(m.taskList(40, 10))
	assert.Contains(t, list, "│  └─ ● ↳ shared")
	assert.Contains(t, list, "   └─ ● ↳ shared")
	assert.NotContains(t, list, "#1 shared")
	assert.NotContains(t, list, "#2 shared")
}

func TestTUIModelMarksOwnerSharedWhenJoinEventArrivesFirst(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m.taskNavigator = taskNavigatorTree
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, scheduled(3, 1, "shared"))
	m = updateTUIModel(t, m, taskJoinedMsg{id: 3, ownerID: 2})
	m = updateTUIModel(t, m, started(2, 1, "shared"))

	assert.True(t, m.byID[2].shared)
	assert.True(t, m.byID[3].shared)
	assert.Equal(t, []uint64{1, 2, 3}, rowIDs(m.taskRows()))
	assert.Equal(t, "#1 shared", m.taskName(m.byID[2]))
	assert.Equal(t, "#2 shared", m.taskName(m.byID[3]))
}

func TestTUIModelNestsExecutionsUnderTheirParent(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m.taskNavigator = taskNavigatorTree
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(5, 0, "other-root"))
	m = updateTUIModel(t, m, started(2, 1, "child"))
	m = updateTUIModel(t, m, startedUnder(3, 2, 1, "grandchild"))
	m = updateTUIModel(t, m, started(4, 1, "second-child"))

	rows := m.taskRows()
	assert.Equal(t, []string{"root", "child", "grandchild", "second-child", "other-root"}, rowNames(rows))
	list := ansi.Strip(m.taskList(30, 10))
	lines := strings.Split(list, "\n")
	require.GreaterOrEqual(t, len(lines), 5)
	assert.True(t, strings.HasPrefix(lines[1], "● root"), lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "├─ ● child"), lines[2])
	assert.True(t, strings.HasPrefix(lines[3], "│  └─ ● grandchild"), lines[3])
	assert.True(t, strings.HasPrefix(lines[4], "└─ ● second-child"), lines[4])
}

func TestTUIModelShowsMultipleIndependentRoots(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, started(2, 1, "compile"))
	m = updateTUIModel(t, m, started(3, 0, "test"))
	m = updateTUIModel(t, m, started(4, 3, "unit"))

	assert.Equal(t, []string{"build", "compile", "test", "unit"}, rowNames(m.taskRows()))
	list := ansi.Strip(m.taskList(40, 10))
	assert.Contains(t, list, "● build\n└─ ● compile")
	assert.Contains(t, list, "● test\n└─ ● unit")
}

func TestTUIModelNumbersRepeatedRootCalls(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, started(2, 0, "build"))

	assert.Equal(t, "#1 build", m.taskName(m.byID[1]))
	assert.Equal(t, "#2 build", m.taskName(m.byID[2]))
	m.selectTask(1)
	m = updateTUIModel(t, m, taskJoinedMsg{id: 2, ownerID: 1})
	assert.Equal(t, []uint64{1, 2}, rowIDs(m.taskRows()))
	assert.Equal(t, uint64(2), m.selectedID)
	assert.Equal(t, uint64(1), m.selectedTask().id)
}

func TestTUIModelSkipsTasksNotAttemptedWhenExecutionEnds(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, scheduled(1, 1, "first"))
	m = updateTUIModel(t, m, started(2, 0, "second"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	m = updateTUIModel(t, m, executionDoneMsg{})

	assert.Equal(t, taskSkipped, m.byID[1].state)
	assert.Equal(t, "○", taskIconText(taskSkipped))
	assert.Equal(t, "skipped", taskStateText(taskSkipped))
	assert.Equal(t, taskSucceeded, m.byID[2].state)
}

func TestTUIModelPrefersFirstChildButAllowsSelectingRoot(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, scheduled(1, 1, "root"))
	assert.True(t, m.hasSelect)
	assert.Equal(t, uint64(1), m.selectedID)
	m = updateTUIModel(t, m, scheduled(2, 1, "child"))
	assert.Equal(t, taskPending, m.byID[2].state)
	assert.Equal(t, uint64(2), m.selectedID)

	m.selectTask(0)
	assert.Equal(t, uint64(1), m.selectedID)
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	assert.Equal(t, taskSucceeded, m.byID[2].state)
}

func TestTUIModelMakesRootOutputAccessible(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "root", data: "root output\n"})

	assert.Equal(t, uint64(1), m.selectedID)
	assert.Contains(t, m.outputPanel(40), "OUTPUT · root")
	assert.Contains(t, m.viewport.View(), "root output")
	assert.Contains(t, ansi.Strip(m.taskList(30, 10)), "● root")
}

func TestTUIModelKeepsRootSelectedWhenItProducedOutputBeforeChild(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "root", data: "root output\n"})
	m = updateTUIModel(t, m, started(2, 1, "child"))

	assert.Equal(t, uint64(1), m.selectedID)
	m.moveSelection(1)
	assert.Equal(t, uint64(2), m.selectedID)
	m.moveSelection(-1)
	assert.Equal(t, uint64(1), m.selectedID)
	assert.Contains(t, m.viewport.View(), "root output")
}

func TestTUIModelListNavigatorFlattensTasksAndCollapsesSharedCalls(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m.taskNavigator = taskNavigatorList
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "parent-a"))
	m = updateTUIModel(t, m, started(3, 1, "parent-b"))
	m = updateTUIModel(t, m, startedUnder(4, 2, 1, "shared"))
	m = updateTUIModel(t, m, scheduledUnder(5, 3, 1, "shared"))
	m.selectTask(4)
	assert.Equal(t, uint64(5), m.selectedID)
	m = updateTUIModel(t, m, taskJoinedMsg{id: 5, ownerID: 4})

	assert.Equal(t, []uint64{1, 2, 3, 4}, rowIDs(m.taskRows()))
	assert.Equal(t, uint64(4), m.selectedID)
	list := ansi.Strip(m.taskList(40, 10))
	assert.Contains(t, list, "├─ ● parent-a")
	assert.Contains(t, list, "├─ ● parent-b")
	assert.Contains(t, list, "└─ ● shared")
	assert.NotContains(t, list, "↳")
	assert.NotContains(t, list, "#1 shared")
}

func TestTUIModelMouseSelectsTasksAndFocusesPanes(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "first"))
	m = updateTUIModel(t, m, started(3, 1, "second"))

	// The title occupies row 1; the root is row 2 and the second child is row 4.
	m = updateTUIModel(t, m, tea.MouseClickMsg{X: 5, Y: 4, Button: tea.MouseLeft})
	assert.Equal(t, uint64(3), m.selectedID)
	assert.Equal(t, taskPane, m.focus)

	layout := newTUILayout(m.width, m.height)
	m = updateTUIModel(t, m, tea.MouseClickMsg{X: layout.leftOuterWidth + layout.gap + 2, Y: 3, Button: tea.MouseLeft})
	assert.Equal(t, outputPane, m.focus)
}

func TestTUIModelFullscreenOutputDisablesMouseAndShowsLiveOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 12})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "first\n"})
	assert.Contains(t, ansi.Strip(m.View().Content), "f: fullscreen")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'f', Text: "f"})
	selectionView := m.View()
	assert.True(t, m.fullscreenOutput)
	assert.Equal(t, tea.MouseModeNone, selectionView.MouseMode)
	assert.Contains(t, ansi.Strip(selectionView.Content), "g/G: top/bottom")
	assert.NotContains(t, ansi.Strip(selectionView.Content), "drag")
	assert.Contains(t, selectionView.Content, "first")
	assert.NotContains(t, selectionView.Content, "TASKS")
	assert.NotContains(t, selectionView.Content, "╭")

	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "second\n"})
	assert.NotEqual(t, selectionView.Content, m.View().Content)
	assert.Contains(t, m.View().Content, "second")
	assert.True(t, m.fullscreenViewport.AtBottom())
	assert.Equal(t, "first\nsecond\n", m.byID[2].output)

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.False(t, m.fullscreenOutput)
	assert.Equal(t, tea.MouseModeCellMotion, m.View().MouseMode)
	assert.Contains(t, m.View().Content, "second")
}

func TestTUIModelFullscreenOutputScrollsWithKeyboard(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 10})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: numberedLines(60)})
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'f', Text: "f"})

	require.True(t, m.fullscreenViewport.AtBottom())
	bottomOffset := m.fullscreenViewport.YOffset()
	require.Greater(t, bottomOffset, 0)

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	scrolledOffset := m.fullscreenViewport.YOffset()
	assert.Less(t, scrolledOffset, bottomOffset)
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "new output\n"})
	assert.Equal(t, scrolledOffset, m.fullscreenViewport.YOffset())
	assert.Contains(t, m.View().Content, "scroll")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'g', Text: "g"})
	assert.True(t, m.fullscreenViewport.AtTop())
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'G', Text: "G"})
	assert.True(t, m.fullscreenViewport.AtBottom())
}

func TestTUIModelScrollsAndRemembersEachTaskOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 10})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "first"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "first", data: numberedLines(30)})
	m = updateTUIModel(t, m, started(3, 1, "second"))
	m.focus = outputPane
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})

	firstOffset := m.viewport.YOffset()
	assert.Greater(t, firstOffset, 0)
	assert.False(t, m.byID[2].followOutput)

	m.selectTask(2)
	m.selectTask(1)
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
	require.Nil(t, cmd)
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	m = next.(tuiModel)
	assert.True(t, m.quitting)
	assert.Contains(t, m.View().Content, "waiting for processes to exit")

	next, cmd = m.Update(executionDoneMsg{})
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
	assert.True(t, next.(tuiModel).done)
}

func TestTUIModelBackCancelsBeforeReturningToLauncher(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	m := newTUIModel(cancel)
	m.canReturnToLauncher = true
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Nil(t, cmd)
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	m = next.(tuiModel)
	assert.True(t, m.returning)
	assert.Contains(t, m.View().Content, "returning to launcher")

	next, cmd = m.Update(executionDoneMsg{})
	require.NotNil(t, cmd)
	assert.IsType(t, returnToLauncherMsg{}, cmd())
	assert.True(t, next.(tuiModel).done)
}

func TestTUIOutputQueueCoalescesWrites(t *testing.T) {
	t.Parallel()

	tui := &UI{pending: make(map[uint64]pendingOutput)}
	tui.enqueueOutput(7, "build", "one")
	tui.enqueueOutput(7, "build", " two")

	assert.Equal(t, map[uint64]pendingOutput{7: {name: "build", data: "one two"}}, tui.drainOutput())
	assert.False(t, tui.outputQueued)
}

func started(id, rootID uint64, name string) taskStartedMsg {
	if rootID == 0 {
		rootID = id
	}
	parentID := rootID
	if id == rootID {
		parentID = 0
	}
	return startedUnder(id, parentID, rootID, name)
}

func startedUnder(id, parentID, rootID uint64, name string) taskStartedMsg {
	return taskStartedMsg{task: taskInvocation{ID: id, ParentID: parentID, RootID: rootID, Name: name}}
}

func scheduled(id, rootID uint64, name string) taskScheduledMsg {
	parentID := rootID
	if id == rootID {
		parentID = 0
	}
	return scheduledUnder(id, parentID, rootID, name)
}

func scheduledUnder(id, parentID, rootID uint64, name string) taskScheduledMsg {
	return taskScheduledMsg{task: taskInvocation{ID: id, ParentID: parentID, RootID: rootID, Name: name}}
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

func rowIDs(rows []tuiTaskRow) []uint64 {
	ids := make([]uint64, len(rows))
	for i, row := range rows {
		ids[i] = row.task.id
	}
	return ids
}

func numberedLines(count int) string {
	var output string
	for i := range count {
		output += fmt.Sprintf("line %02d\n", i)
	}
	return output
}

func TestRunRejectsUnknownTasksWithoutOpeningTheTUI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := "version: '3'\ntasks:\n  build: echo built\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o600))

	var screen bytes.Buffer
	log := &logger.Logger{
		AssumeTerm: true,
		Stdin:      strings.NewReader(""),
		Stdout:     &screen,
		Stderr:     &screen,
	}
	ui, err := New(log, Options{})
	require.NoError(t, err)

	e := task.NewExecutor(task.WithDir(dir), task.WithStdout(io.Discard), task.WithStderr(io.Discard))
	require.NoError(t, e.Setup())

	err = ui.Run(t.Context(), e, []*task.Call{{Task: "nope"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope")
	assert.Empty(t, screen.String(), "the terminal must be untouched when the task cannot be resolved")
	assert.Nil(t, e.Listener, "the executor must not be left with a listener attached")
}

func TestTUIModelShowsSkippedCallsAsSkipped(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, scheduledUnder(2, 1, 1, "other-platform"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2, err: errTaskSkipped})

	assert.Equal(t, taskSkipped, m.byID[2].state)
	// A skipped call is not a failure, so its error is not written to its output.
	assert.Empty(t, m.byID[2].output)
}

func TestTUIModelShowsCallsThatNeverCompiled(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, scheduledUnder(2, 1, 1, "typoo"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2, err: errors.New(`task: Task "typoo" does not exist`)})

	assert.Equal(t, taskFailed, m.byID[2].state)
	assert.Contains(t, rowNames(m.taskRows()), "typoo")
	assert.Contains(t, m.byID[2].output, `Task "typoo" does not exist`)
}

func TestTUIModelRenamesCallsOnceCompilationResolvesTheName(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	// Announced under the raw Taskfile name, then started under its label.
	m = updateTUIModel(t, m, scheduledUnder(2, 1, 1, "docs"))
	assert.Contains(t, rowNames(m.taskRows()), "docs")

	m = updateTUIModel(t, m, startedUnder(2, 1, 1, "Build the docs"))
	assert.Contains(t, rowNames(m.taskRows()), "Build the docs")
	assert.NotContains(t, rowNames(m.taskRows()), "docs")
}

func TestTUIOutputRedrawsLinesOnCarriageReturn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "a progress bar collapses to its last frame",
			parts: []string{"Downloading  0%\rDownloading 50%\rDownloading 100%\nDone\n"},
			want:  "Downloading 100%\nDone\n",
		},
		{
			name:  "a redraw arriving in a later write replaces the line",
			parts: []string{"Downloading  0%", "\rDownloading 50%", "\rDownloading 100%\n"},
			want:  "Downloading 100%\n",
		},
		{
			name:  "earlier complete lines survive a redraw",
			parts: []string{"building\nDownloading  0%\rDownloading 100%\n"},
			want:  "building\nDownloading 100%\n",
		},
		{
			name:  "a trailing carriage return leaves the line visible",
			parts: []string{"partial\r"},
			want:  "partial",
		},
		{
			name:  "a carriage return followed by a newline keeps the line",
			parts: []string{"kept\r", "\nnext\n"},
			want:  "kept\nnext\n",
		},
		{
			name:  "a redraw spanning two writes replaces only the current line",
			parts: []string{"first\nsecond\r", "third\n"},
			want:  "first\nthird\n",
		},
		{
			name:  "windows line endings stay line breaks",
			parts: []string{"first\r\nsecond\r\n"},
			want:  "first\nsecond\n",
		},
		{
			name:  "output without carriage returns is untouched",
			parts: []string{"one\n", "two\n"},
			want:  "one\ntwo\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			m := newTUIModel(func() {})
			m = updateTUIModel(t, m, started(1, 0, "build"))
			for _, part := range test.parts {
				m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: part})
			}
			assert.Equal(t, test.want, m.byID[1].output)
		})
	}
}

func TestTrimPartialRuneKeepsOutputValid(t *testing.T) {
	t.Parallel()

	// Slicing the output buffer at a fixed byte length can land inside a rune.
	const text = "héllo"
	for cut := range len(text) + 1 {
		got := trimPartialRune(text[cut:])
		assert.True(t, utf8.ValidString(got), "cut at %d produced %q", cut, got)
		assert.True(t, strings.HasSuffix(text, got), "cut at %d dropped too much: %q", cut, got)
	}
	assert.Equal(t, "llo", trimPartialRune(text[3:]), "the half of é must be dropped")
}

func TestTUIModelCopiesSelectedOutputToClipboard(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: "compiling\n"})

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m = next.(tuiModel)
	require.NotNil(t, cmd)

	// The notice reports the outcome of the copy, not the attempt.
	m = updateTUIModel(t, m, clipboardCopiedMsg{size: 10, confirmed: true})
	assert.Contains(t, m.View().Content, "copied 10 B")
	assert.NotContains(t, m.View().Content, "press s")

	// The notice clears itself, and a stale timer must not clear a newer one.
	m.noticeID++
	m = updateTUIModel(t, m, noticeExpiredMsg{id: m.noticeID - 1})
	assert.NotEmpty(t, m.notice, "an outdated timer must not clear the current notice")
	m = updateTUIModel(t, m, noticeExpiredMsg{id: m.noticeID})
	assert.Empty(t, m.notice)
}

func TestTUIModelReportsWhenThereIsNothingToCopy(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m = next.(tuiModel)
	require.NotNil(t, cmd)
	assert.Contains(t, m.View().Content, "nothing to copy")
}

func TestSnapshotOutputBody(t *testing.T) {
	t.Parallel()

	finished := (&snapshotOutput{text: "compiling\n"}).body()
	assert.Contains(t, finished, "compiling\n")
	assert.Contains(t, finished, "Press Enter to return")
	assert.NotContains(t, finished, "still running")

	running := (&snapshotOutput{text: "compiling", running: true}).body()
	assert.Contains(t, running, "still running")
	// A body that did not end in a newline must not run into the footer.
	assert.Contains(t, running, "compiling\n")

	empty := (&snapshotOutput{}).body()
	assert.Contains(t, empty, "(no output)")
}

func TestSnapshotOutputWaitsForEnter(t *testing.T) {
	t.Parallel()

	var screen bytes.Buffer
	snapshot := &snapshotOutput{text: "hello\n"}
	snapshot.SetStdout(&screen)
	snapshot.SetStdin(strings.NewReader("\n"))

	require.NoError(t, snapshot.Run())
	assert.Contains(t, screen.String(), "hello")
	assert.Contains(t, screen.String(), "Press Enter to return")
}

func TestHumanizeBytes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "12 B", humanizeBytes(12))
	assert.Equal(t, "1.0 KB", humanizeBytes(1024))
	assert.Equal(t, "1.5 MB", humanizeBytes(1024*1024*3/2))
}

func TestTUIModelAdmitsWhenAClipboardCopyCannotBeConfirmed(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: "compiling\n"})

	// No clipboard helper ran, so only OSC 52 was sent. It has no reply, and
	// VTE-based terminals discard it, so the notice must not claim success.
	m = updateTUIModel(t, m, clipboardCopiedMsg{size: 10})
	assert.Contains(t, m.View().Content, "press s")
}

func TestSystemClipboardArgsPrefersTheSessionsTool(t *testing.T) {
	// Not parallel: it sets environment variables.
	dir := t.TempDir()
	for _, name := range []string{"wl-copy", "xclip"} {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700))
	}
	t.Setenv("PATH", dir)

	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", "")
	args, ok := systemClipboardArgs()
	require.True(t, ok)
	assert.Equal(t, "wl-copy", args[0])

	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", ":0")
	args, ok = systemClipboardArgs()
	require.True(t, ok)
	assert.Equal(t, []string{"xclip", "-selection", "clipboard"}, args)
}

func TestCopyToSystemClipboardReportsWhenNoHelperExists(t *testing.T) {
	// Not parallel: it sets environment variables.
	t.Setenv("PATH", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")

	msg, ok := copyToSystemClipboard("hello", false)().(clipboardCopiedMsg)
	require.True(t, ok)
	assert.Equal(t, 5, msg.size)
	assert.False(t, msg.confirmed, "no helper ran, so the copy cannot be confirmed")
}

func TestTUIModelCopiesWithoutColourCodes(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{
		id:   1,
		name: "build",
		data: "\x1b[31mFAILED\x1b[0m: two tests\n",
	})

	// The pane keeps the colours; the clipboard gets the characters, which is
	// what selecting the same text in a terminal would give.
	assert.Contains(t, m.byID[1].output, "\x1b[31m")

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	require.NotNil(t, cmd)

	copied := copyText(m.byID[1].output, false)
	assert.Equal(t, "FAILED: two tests\n", copied)
	assert.NotContains(t, copied, "\x1b")
}

func TestTUIModelCopiesWithColoursOnShiftY(t *testing.T) {
	t.Parallel()

	const coloured = "\x1b[31mFAILED\x1b[0m\n"
	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: coloured})

	assert.Equal(t, "FAILED\n", copyText(m.byID[1].output, false))
	assert.Equal(t, coloured, copyText(m.byID[1].output, true), "Y must keep the escape sequences")

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'Y', Text: "Y"})
	require.NotNil(t, cmd)

	// The notice distinguishes the two, so the key teaches itself on use.
	m = updateTUIModel(t, m, clipboardCopiedMsg{size: 7, confirmed: true, colours: true})
	assert.Contains(t, m.View().Content, "with colours")
	m = updateTUIModel(t, m, clipboardCopiedMsg{size: 7, confirmed: true})
	assert.NotContains(t, m.View().Content, "with colours")
}
