package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/interp"

	"github.com/go-task/task/v3"
	taskerrors "github.com/go-task/task/v3/errors"
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
	m = updateTUIModel(t, m, taskFinishedMsg{id: 3, result: resultFailed, err: errors.New("failed")})

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
	m = updateTUIModel(t, m, taskFinishedMsg{id: 5, result: resultFailed, err: errors.New("failed")})
	m = updateTUIModel(t, m, started(6, 1, "canceled-task"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 6, result: resultCanceled})

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

	// Skip the pane header, which carries the state of the run as a whole.
	taskRows := func(m tuiModel) string {
		lines := strings.SplitN(ansi.Strip(m.taskList(30, 10)), "\n", 2)
		require.Len(t, lines, 2)
		return lines[1]
	}

	icons := taskRows(m)
	assert.Contains(t, icons, "└─ ● worker")
	assert.NotContains(t, icons, "running")
	m.statusLabels = true
	labels := taskRows(m)
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
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2, result: resultFailed, err: errors.New("failed")})
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
	list := taskListWithoutDurations(t, m, 40, 10)
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
	assert.Contains(t, ansi.Strip(m.View().Content), "f fullscreen")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'f', Text: "f"})
	selectionView := m.View()
	assert.True(t, m.fullscreenOutput)
	assert.Equal(t, tea.MouseModeNone, selectionView.MouseMode)
	assert.Contains(t, ansi.Strip(selectionView.Content), "? help")
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
	assert.Contains(t, ansi.Strip(m.View().Content), "v select")

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
	// The executor timestamps every event, so a hand-built one does too.
	return taskStartedMsg{
		task: taskInvocation{ID: id, ParentID: parentID, RootID: rootID, Task: name, Name: name},
		at:   time.Now(),
	}
}

func scheduled(id, rootID uint64, name string) taskScheduledMsg {
	parentID := rootID
	if id == rootID {
		parentID = 0
	}
	return scheduledUnder(id, parentID, rootID, name)
}

func scheduledUnder(id, parentID, rootID uint64, name string) taskScheduledMsg {
	return taskScheduledMsg{task: taskInvocation{ID: id, ParentID: parentID, RootID: rootID, Task: name, Name: name}}
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
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2, result: resultSkipped})

	assert.Equal(t, taskSkipped, m.byID[2].state)
	// A skipped call is not a failure, so its error is not written to its output.
	assert.Empty(t, m.byID[2].output)
}

func TestTUIModelShowsCallsThatNeverCompiled(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, scheduledUnder(2, 1, 1, "typoo"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2, result: resultFailed, err: errors.New(`task: Task "typoo" does not exist`)})

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

	finished := (&snapshotOutput{name: "build", text: "compiling\n", width: 40}).body()
	assert.Contains(t, finished, "compiling\n")
	// The dump carries no other context, so it always says what it is.
	assert.Contains(t, finished, "snapshot: build")
	assert.Contains(t, finished, "end of snapshot")
	assert.Contains(t, finished, "Press Enter to return")
	assert.NotContains(t, finished, "still running")

	running := (&snapshotOutput{name: "build", text: "compiling", running: true, width: 40}).body()
	assert.Contains(t, ansi.Strip(running), "still running")
	// Output that did not end in a newline must not run into the footer.
	assert.Contains(t, running, "compiling\n")
	// The warning is coloured; the rest of the footer is not.
	assert.NotEqual(t, ansi.Strip(running), running, "the warning must stand out")
	assert.NotContains(t, finished, "\x1b", "a finished snapshot needs no colour")

	empty := (&snapshotOutput{name: "build", width: 40}).body()
	assert.Contains(t, empty, "(no output)")
}

func TestSnapshotOutputStartsOnABlankScreen(t *testing.T) {
	t.Parallel()

	snapshot := &snapshotOutput{name: "build", text: "hi\n", width: 40, height: 24}
	blank := snapshot.blankScreen()

	// Scrolling the old screen away keeps it in scrollback; erasing it might
	// not, depending on the terminal.
	assert.Equal(t, 24, strings.Count(blank, "\n"))
	assert.True(t, strings.HasSuffix(blank, "\x1b[H"), "cursor must return to the top")
	assert.NotContains(t, blank, "2J", "the screen must not be erased")

	// A zero height means we do not know the terminal size; print nothing.
	assert.Empty(t, (&snapshotOutput{}).blankScreen())
}

func TestSnapshotOutputWaitsForEnter(t *testing.T) {
	t.Parallel()

	var screen bytes.Buffer
	snapshot := &snapshotOutput{name: "build", text: "hello\n", width: 40}
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
	assert.Contains(t, m.View().Content, "press t")
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

func TestTUIModelShowsRunStateInTheTaskPaneHeader(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	header := func(m tuiModel) string {
		return strings.SplitN(ansi.Strip(m.taskList(40, 10)), "\n", 2)[0]
	}
	assert.Contains(t, header(m), "running")

	done := updateTUIModel(t, m, executionDoneMsg{})
	assert.Contains(t, header(done), "complete")

	failed := updateTUIModel(t, m, executionDoneMsg{err: errors.New("boom")})
	assert.Contains(t, header(failed), "failed")

	// The footer stays dedicated to keys.
	assert.NotContains(t, ansi.Strip(done.View().Content), "execution complete")
}

func TestTUIModelOpensAndClosesTheKeyList(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m.canReturnToLauncher = true
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 24})
	m = updateTUIModel(t, m, started(1, 0, "build"))

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	require.True(t, m.showHelp)
	page := ansi.Strip(m.View().Content)
	// Everything the short footer had no room for must be listed here.
	for _, expected := range []string{"Y", "copy output with ANSI codes", "wheel", "click", "launcher", "quit"} {
		assert.Contains(t, page, expected)
	}
	assert.Equal(t, tea.MouseModeNone, m.View().MouseMode)

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	assert.False(t, m.showHelp, "any key returns from the key list")
}

func TestTUIViewsFitTheTerminal(t *testing.T) {
	t.Parallel()

	for _, size := range []struct{ width, height int }{{40, 10}, {80, 24}, {200, 60}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			t.Parallel()
			base := newTUIModel(func() {})
			base = updateTUIModel(t, base, tea.WindowSizeMsg{Width: size.width, Height: size.height})
			base = updateTUIModel(t, base, started(1, 0, "a-task-with-a-fairly-long-name"))

			views := map[string]tuiModel{
				"dashboard":  base,
				"fullscreen": updateTUIModel(t, base, tea.KeyPressMsg{Code: 'f', Text: "f"}),
				"keys":       updateTUIModel(t, base, tea.KeyPressMsg{Code: '?', Text: "?"}),
			}
			for name, m := range views {
				content := m.View().Content
				assert.LessOrEqual(t, lipgloss.Width(content), size.width, "%s is too wide", name)
				assert.LessOrEqual(t, lipgloss.Height(content), size.height, "%s is too tall", name)
			}
		})
	}
}

func TestDashboardKeysHideTheLauncherWhenThereIsNoneToReturnTo(t *testing.T) {
	t.Parallel()

	withLauncher := newDashboardKeys(false, true)
	assert.True(t, withLauncher.Launcher.Enabled())

	direct := newDashboardKeys(false, false)
	assert.False(t, direct.Launcher.Enabled(), "a disabled binding is left out of the help")
}

func TestDashboardKeysDescribeArrowsByFocus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "select a task", newDashboardKeys(false, true).Move.Help().Desc)
	assert.Equal(t, "scroll the output", newDashboardKeys(true, true).Move.Help().Desc)

	// The footer restates them in a word.
	shortDesc := func(outputFocused bool) string {
		for _, binding := range newDashboardKeys(outputFocused, true).ShortHelp() {
			if binding.Help().Key == "↑/↓" {
				return binding.Help().Desc
			}
		}
		return ""
	}
	assert.Equal(t, "select", shortDesc(false))
	assert.Equal(t, "scroll", shortDesc(true))
}

func TestShortHelpKeepsTheWayOutOnANarrowTerminal(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	bindings := newDashboardKeys(false, true).ShortHelp()
	for _, width := range []int{20, 40, 60, 80, 200} {
		line := shortHelp(m.help, bindings, width)
		assert.LessOrEqual(t, lipgloss.Width(line), width, "footer overflows at %d", width)
		if width >= 20 {
			// Help and quit lead, so truncation eats the tail rather than the
			// way out of the view.
			assert.Contains(t, ansi.Strip(line), "? help", "at %d columns", width)
			assert.Contains(t, ansi.Strip(line), "q quit", "at %d columns", width)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	// A quick task reports milliseconds rather than nothing, so every row that
	// ran carries a number.
	assert.Equal(t, "0ms", formatDuration(0))
	assert.Equal(t, "3ms", formatDuration(3*time.Millisecond))
	assert.Equal(t, "400ms", formatDuration(400*time.Millisecond))
	assert.Equal(t, "3.4s", formatDuration(3400*time.Millisecond))
	assert.Equal(t, "12s", formatDuration(12*time.Second))
	assert.Equal(t, "1m35s", formatDuration(95*time.Second))
	assert.Equal(t, "2h05m", formatDuration(125*time.Minute))
}

func TestTUIModelShowsHowLongTasksTook(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, startedUnder(2, 1, 1, "compile"))

	now := time.Now()
	m.byID[1].startedAt = now.Add(-95 * time.Second)
	m.byID[2].startedAt = now.Add(-3400 * time.Millisecond)
	m.byID[2].finishedAt = now

	pane := ansi.Strip(m.taskList(34, 8))
	assert.Contains(t, pane, "1m35s", "a running task counts up")
	assert.Contains(t, pane, "3.4s", "a finished task keeps its final duration")

	// A narrow pane keeps the names and drops the durations.
	assert.NotContains(t, ansi.Strip(m.taskList(18, 8)), "1m35s")
}

func TestTUIModelTicksOnlyWhileTasksRun(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	next, cmd := m.Update(started(1, 0, "build"))
	m = next.(tuiModel)
	require.NotNil(t, cmd, "a running task schedules a redraw")
	assert.True(t, m.ticking)

	// A second start must not stack a second ticker.
	next, cmd = m.Update(startedUnder(2, 1, 1, "compile"))
	m = next.(tuiModel)
	assert.Nil(t, cmd, "only one ticker at a time")

	// Once everything has finished the ticker stops, so an idle dashboard is
	// completely static.
	m = updateTUIModel(t, m, taskFinishedMsg{id: 1})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	next, cmd = m.Update(elapsedTickMsg{})
	m = next.(tuiModel)
	assert.Nil(t, cmd)
	assert.False(t, m.ticking)
}

func TestTUIModelStopsTheClockOnCancelledTasks(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "slow"))
	m = updateTUIModel(t, m, executionDoneMsg{})

	require.Equal(t, taskCanceled, m.byID[1].state)
	assert.False(t, m.byID[1].finishedAt.IsZero(), "a cancelled task must stop counting up")
}

func TestFullHelpUsesTheColumnsThatFit(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	bindings := newDashboardKeys(false, true).allBindings()

	countColumns := func(width int) int {
		view := fullHelp(m.help, bindings, width)
		require.LessOrEqual(t, lipgloss.Width(view), width, "key list overflows at %d", width)
		// Every binding is listed whatever the layout.
		for _, binding := range bindings {
			assert.Contains(t, ansi.Strip(view), binding.Help().Desc)
		}
		return len(strings.Split(strings.TrimRight(ansi.Strip(view), "\n"), "\n"))
	}

	// Narrower means taller: the descriptions are written to be read, so the
	// layout gives way rather than the wording.
	wide, narrow := countColumns(140), countColumns(50)
	assert.Less(t, wide, narrow, "a narrow terminal should stack into more rows")
}

func TestFooterKeepsTheWayOutAtEightyColumns(t *testing.T) {
	t.Parallel()

	// The whole line is not expected to fit eighty columns; it is ordered so
	// that what does fit is what a reader needs to get somewhere else.
	m := newTUIModel(func() {})
	// The arrow keys are deliberately last: they are the part of a TUI a reader
	// can guess, so they are what an eighty column terminal gives up.
	dashboard := ansi.Strip(shortHelp(m.help, newDashboardKeys(false, true).ShortHelp(), 80))
	for _, expected := range []string{"? help", "q quit", "esc/b launcher", "y copy", "t to terminal"} {
		assert.Contains(t, dashboard, expected, "footer at 80 columns: %s", dashboard)
	}

	full := ansi.Strip(shortHelp(m.help, newFullscreenKeys(false).ShortHelp(), 80))
	for _, expected := range []string{"? help", "q quit", "f/esc back"} {
		assert.Contains(t, full, expected, "fullscreen footer at 80 columns: %s", full)
	}

	// Truncation drops whole entries: a line that was cut ends at an entry
	// boundary followed by the ellipsis, never part-way through a word.
	for _, width := range []int{40, 60, 80, 100} {
		line := ansi.Strip(shortHelp(m.help, newDashboardKeys(false, true).ShortHelp(), width))
		assert.LessOrEqual(t, lipgloss.Width(line), width)
		if strings.HasSuffix(line, "…") {
			// A trimmed line ends at an entry boundary, never part-way through a
			// word and never on a dangling separator.
			assert.True(t, strings.HasSuffix(line, " …"),
				"at %d columns the line was cut mid-entry: %s", width, line)
			assert.NotContains(t, line, "• …",
				"at %d columns the line ends on a separator: %s", width, line)
		}
	}
}

func TestPrintToTerminalIsBoundToT(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: "hi\n"})

	_, cmd := m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	assert.NotNil(t, cmd, "t prints the output to the terminal")

	// s used to mean snapshot, before printing moved to t. It now saves.
	keys := newDashboardKeys(false, true)
	assert.Equal(t, "print output to terminal", keys.Snapshot.Help().Desc)
	assert.Equal(t, "save output to a file", keys.Save.Help().Desc)
}

func TestFooterPairsTheArrowKeys(t *testing.T) {
	t.Parallel()

	// Vertical arrows move the selection and horizontal arrows move panes, so
	// the two entries sit next to each other rather than either being described
	// as tab.
	bindings := newDashboardKeys(false, true).ShortHelp()
	var keys []string
	for _, binding := range bindings {
		keys = append(keys, binding.Help().Key)
	}
	require.Len(t, keys, 9)
	assert.Equal(t, []string{"↑/↓", "←/→"}, keys[len(keys)-2:], "the arrows are adjacent and last")
	assert.Equal(t, "pane", bindings[len(bindings)-1].Help().Desc)
}

// taskListWithoutDurations renders the task pane with the right-aligned
// duration column trimmed, for assertions about names and tree structure.
func taskListWithoutDurations(t *testing.T, m tuiModel, width, height int) string {
	t.Helper()
	var lines []string
	for line := range strings.SplitSeq(ansi.Strip(m.taskList(width, height)), "\n") {
		lines = append(lines, strings.TrimRight(regexp.
			MustCompile(`\s+\d[\dhms.]*$`).
			ReplaceAllString(line, ""), " "))
	}
	return strings.Join(lines, "\n")
}

func TestTUIModelReportsNoDurationForTasksThatNeverRan(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, scheduledUnder(2, 1, 1, "never-attempted"))

	// A task that ran has a duration even if it was instant; one that never
	// started has none, which is different from a duration of zero. Pin the
	// start so the assertion does not depend on how long the test itself took.
	m.byID[1].startedAt = time.Now()
	m.byID[1].finishedAt = m.byID[1].startedAt
	assert.Equal(t, "0ms", m.durationLabel(m.byID[1]))
	assert.Empty(t, m.durationLabel(m.byID[2]))
	assert.NotContains(t, ansi.Strip(m.taskList(40, 10)), "never-attempted   0ms")
}

func TestPromptAsksTheUserAndReturnsTheAnswer(t *testing.T) {
	t.Parallel()

	press := func(m tuiModel, keys ...string) tuiModel {
		for _, key := range keys {
			next, _ := m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
			m = next.(tuiModel)
		}
		return m
	}

	t.Run("confirm", func(t *testing.T) {
		t.Parallel()
		m := newTUIModel(func() {})
		state := &promptState{kind: promptConfirm, task: "deploy", message: "Really?", done: make(chan promptAnswer, 1)}
		m.beginPrompt(state)
		assert.Contains(t, ansi.Strip(m.View().Content), "Really?")

		m = press(m, "y")
		assert.Equal(t, promptAnswer{confirmed: true}, <-state.done)
		assert.Nil(t, m.prompt, "the question leaves the screen once answered")
	})

	t.Run("declining is not an error", func(t *testing.T) {
		t.Parallel()
		m := newTUIModel(func() {})
		state := &promptState{kind: promptConfirm, done: make(chan promptAnswer, 1)}
		m.beginPrompt(state)

		m = press(m, "n")
		answer := <-state.done
		assert.False(t, answer.confirmed)
		assert.NoError(t, answer.err, "declining stops the task without being an error")
	})

	t.Run("free text", func(t *testing.T) {
		t.Parallel()
		m := newTUIModel(func() {})
		state := &promptState{kind: promptText, name: "RELEASE_NAME", done: make(chan promptAnswer, 1)}
		m.beginPrompt(state)

		m = press(m, "v", "1")
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
		m = next.(tuiModel)
		assert.Equal(t, "v1", (<-state.done).value)
	})

	t.Run("choice", func(t *testing.T) {
		t.Parallel()
		m := newTUIModel(func() {})
		state := &promptState{
			kind:    promptChoice,
			name:    "ENVIRONMENT",
			options: []string{"development", "staging", "production"},
			done:    make(chan promptAnswer, 1),
		}
		m.beginPrompt(state)
		assert.Contains(t, ansi.Strip(m.View().Content), "staging")

		m = press(m, "j")
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
		m = next.(tuiModel)
		assert.Equal(t, "staging", (<-state.done).value)
	})

	t.Run("cancelling a value stops the run", func(t *testing.T) {
		t.Parallel()
		m := newTUIModel(func() {})
		state := &promptState{kind: promptText, done: make(chan promptAnswer, 1)}
		m.beginPrompt(state)

		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc, Text: "esc"})
		m = next.(tuiModel)
		assert.ErrorIs(t, (<-state.done).err, task.ErrPromptCancelled)
	})
}

func TestPromptRefusesAVariableTypeItCannotRender(t *testing.T) {
	t.Parallel()

	// Task can add variable types; guessing would produce a value the task acts
	// on, so an unknown one is refused.
	ui := &UI{}
	_, err := ui.Ask(task.VarRequest{Task: "deploy", Name: "COUNT", Type: nil})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot ask")
}

func TestTUIModelShowsAFailureThatBelongsToNoTask(t *testing.T) {
	t.Parallel()

	// Declining a prompt fails the run before anything is scheduled. Without
	// somewhere to put it, the dashboard would say the run failed while showing
	// an empty task list and no reason.
	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	m = updateTUIModel(t, m, executionDoneMsg{err: errors.New(`task: task "deploy" cancelled by user`)})

	pane := ansi.Strip(m.View().Content)
	assert.Contains(t, pane, systemTaskName)
	assert.Contains(t, pane, "cancelled by user")
	assert.NotContains(t, pane, "Waiting for tasks")
}

func TestTUIModelDoesNotRepeatAFailureAlreadyShownAgainstATask(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 1, result: resultFailed, err: errors.New("exit status 1")})
	m = updateTUIModel(t, m, executionDoneMsg{err: errors.New(`task: Failed to run task "build": exit status 1`)})

	// The task carries its own failure, so the run's error is not repeated.
	assert.NotContains(t, rowNames(m.taskRows()), systemTaskName)
}

func TestPromptControlCQuitsTheInterface(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	state := &promptState{kind: promptText, name: "RELEASE_NAME", done: make(chan promptAnswer, 1)}
	m.beginPrompt(state)

	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl, Text: ""})
	m = next.(tuiModel)

	// The waiting task is released first, or it would never return.
	assert.ErrorIs(t, (<-state.done).err, task.ErrPromptCancelled)
	assert.True(t, m.quitting, "ctrl+c closes the interface, as it does elsewhere")
	assert.Nil(t, m.prompt)
}

func TestPromptIsADialogOverTheDashboard(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 90, Height: 14})
	m = updateTUIModel(t, m, started(1, 0, "release"))

	m.beginPrompt(&promptState{
		kind: promptChoice, task: "release", name: "ENVIRONMENT",
		options: []string{"staging", "production"}, done: make(chan promptAnswer, 1),
	})
	view := ansi.Strip(m.View().Content)

	// A question is an interruption, not a place you navigated to, so it is
	// drawn over the interface rather than replacing it.
	assert.Contains(t, view, `Task "release" is asking`)
	assert.Contains(t, view, "staging")
	assert.Contains(t, view, "TASKS", "the dashboard stays behind the dialog")
	// The dialog carries its own keys, and the interface behind it offers
	// none, since none of them would do anything.
	assert.Contains(t, view, "enter confirm")
	assert.NotContains(t, view, "y copy")
	assert.NotContains(t, view, "? help")
}

func TestPromptDialogFitsASmallTerminal(t *testing.T) {
	t.Parallel()

	options := make([]string, 30)
	for i := range options {
		options[i] = fmt.Sprintf("option-%02d", i)
	}
	for _, size := range []struct{ width, height int }{{40, 10}, {80, 24}, {200, 60}} {
		m := newTUIModel(func() {})
		m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: size.width, Height: size.height})
		m.beginPrompt(&promptState{
			kind: promptChoice, task: "release", name: "ENVIRONMENT",
			options: options, done: make(chan promptAnswer, 1),
		})
		content := m.View().Content
		assert.LessOrEqual(t, lipgloss.Width(content), size.width, "%dx%d", size.width, size.height)
		assert.LessOrEqual(t, lipgloss.Height(content), size.height, "%dx%d", size.width, size.height)
	}
}

func TestPromptWrapsALongMessage(t *testing.T) {
	t.Parallel()

	long := "This will delete every artifact in the production bucket and cannot be undone"
	wrapped := wrapText(long, 30)

	require.Greater(t, len(strings.Split(wrapped, "\n")), 1, "a long message wraps")
	for line := range strings.SplitSeq(wrapped, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 30)
	}
	// Wrapping breaks between words, not through them.
	assert.Equal(t, strings.Fields(long), strings.Fields(wrapped))
}

func TestConfirmationShowsAndUsesItsDefault(t *testing.T) {
	t.Parallel()

	newConfirm := func() (tuiModel, *promptState) {
		m := newTUIModel(func() {})
		m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 14})
		state := &promptState{kind: promptConfirm, task: "release", message: "Continue?", done: make(chan promptAnswer, 1)}
		m.beginPrompt(state)
		return m, state
	}
	press := func(m tuiModel, code rune, text string) tuiModel {
		next, _ := m.Update(tea.KeyPressMsg{Code: code, Text: text})
		return next.(tuiModel)
	}

	t.Run("the default is marked rather than encoded", func(t *testing.T) {
		t.Parallel()
		_, state := newConfirm()

		lines := strings.Split(strings.TrimPrefix(promptOptions(state, 20), "\n"), "\n")
		require.Len(t, lines, 2)
		assert.Equal(t, []string{"yes", "no"}, []string{
			strings.TrimSpace(ansi.Strip(lines[0])),
			strings.TrimSpace(ansi.Strip(lines[1])),
		})

		// "no" is highlighted, so the answer Enter would give can be seen,
		// rather than hidden in the capitalisation of "[y/N]".
		assert.Equal(t, "  yes", lines[0], "the answer that is not the default is plain")
		assert.NotEqual(t, ansi.Strip(lines[1]), lines[1], "the default is styled")
	})

	t.Run("enter takes the default", func(t *testing.T) {
		t.Parallel()
		m, state := newConfirm()
		press(m, tea.KeyEnter, "enter")
		answer := <-state.done
		assert.False(t, answer.confirmed, "the default is no, as it is on the terminal")
		assert.NoError(t, answer.err)
	})

	t.Run("enter takes yes once it is chosen", func(t *testing.T) {
		t.Parallel()
		m, state := newConfirm()
		m = press(m, 'k', "k")
		press(m, tea.KeyEnter, "enter")
		assert.True(t, (<-state.done).confirmed)
	})

	t.Run("y and n still answer directly", func(t *testing.T) {
		t.Parallel()
		m, state := newConfirm()
		press(m, 'y', "y")
		assert.True(t, (<-state.done).confirmed)

		m, state = newConfirm()
		press(m, 'n', "n")
		assert.False(t, (<-state.done).confirmed)
	})

	t.Run("every key that answers is listed", func(t *testing.T) {
		t.Parallel()
		m, _ := newConfirm()
		view := ansi.Strip(m.View().Content)
		for _, expected := range []string{"enter confirm", "y/n answer", "esc no"} {
			assert.Contains(t, view, expected)
		}
	})
}

func TestFinishedDashboardIsOnlyClosedByADocumentedKey(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 1})
	m = updateTUIModel(t, m, executionDoneMsg{})
	require.True(t, m.done)

	// Enter used to close the finished dashboard without appearing among the
	// keys. It is a confirm key in the dialogs, so pressing it out of habit
	// threw away output the run had left to read.
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
	assert.Nil(t, cmd, "enter does not close the interface")

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	assert.NotNil(t, cmd, "q, which the footer lists, does")
}

// typePath enters a path into the footer field and presses Enter.
func typePath(t *testing.T, m tuiModel, path string) (tuiModel, savedMsg) {
	t.Helper()
	for _, r := range path {
		next, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = next.(tuiModel)
	}
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
	m = next.(tuiModel)
	require.Nil(t, m.save, "the field closes once a path is given")
	require.NotNil(t, cmd)
	saved, ok := cmd().(savedMsg)
	require.True(t, ok)
	return m, saved
}

func withOutput(t *testing.T) tuiModel {
	t.Helper()
	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 90, Height: 20})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: "\x1b[31mFAILED\x1b[0m\n"})
	return m
}

// clearField empties the pre-filled suggestion so a test can type its own path.
func clearField(t *testing.T, m tuiModel) tuiModel {
	t.Helper()
	m.save.input.SetValue("")
	return m
}

func TestSaveAsksWhereToPutTheOutput(t *testing.T) { // nolint:paralleltest // t.Chdir cannot be used in a parallel test
	t.Chdir(t.TempDir())

	m := withOutput(t)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	require.NotNil(t, m.save, "s asks where rather than choosing for the user")

	// The field is filled in, so Enter alone is enough.
	view := ansi.Strip(m.View().Content)
	assert.Contains(t, view, "Save to:")
	assert.Contains(t, view, "build.", "the suggestion leads with the task")
	assert.Contains(t, view, ".log")
	assert.Contains(t, view, "logs", "the default is a logs folder beside the project")
	assert.NotContains(t, view, "~", "not a folder shared by every project")
	assert.Contains(t, view, "enter save")
	// The dashboard stays visible: this is a footer field, not a dialog.
	assert.Contains(t, view, "TASKS")

	m, saved := typePath(t, m, "")
	require.NoError(t, saved.err)
	content, err := os.ReadFile(saved.path)
	require.NoError(t, err)
	assert.Equal(t, "\x1b[31mFAILED\x1b[0m\n", string(content))
}

func TestSaveCreatesMissingDirectories(t *testing.T) { // nolint:paralleltest // t.Chdir cannot be used in a parallel test
	dir := t.TempDir()
	t.Chdir(dir)

	m := withOutput(t)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	m = clearField(t, m)

	_, saved := typePath(t, m, "logs/today/build.log")
	require.NoError(t, saved.err)
	assert.FileExists(t, filepath.Join(dir, "logs", "today", "build.log"))
}

func TestSaveReportsAPathItCannotWrite(t *testing.T) { // nolint:paralleltest // t.Chdir cannot be used in a parallel test
	dir := t.TempDir()
	t.Chdir(dir)
	readOnly := filepath.Join(dir, "read-only")
	require.NoError(t, os.Mkdir(readOnly, 0o500))

	m := withOutput(t)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	m = clearField(t, m)

	m, saved := typePath(t, m, filepath.Join(readOnly, "nested", "build.log"))
	require.Error(t, saved.err, "a directory that cannot be written is reported, not ignored")

	m = updateTUIModel(t, m, saved)
	assert.Contains(t, ansi.Strip(m.View().Content), "save failed")
	assert.Contains(t, ansi.Strip(m.View().Content), "permission denied")
}

func TestSaveCanBeCancelled(t *testing.T) {
	t.Parallel()

	m := withOutput(t)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	require.NotNil(t, m.save)

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyEsc, Text: "esc"})
	assert.Nil(t, m.save)
	assert.Contains(t, ansi.Strip(m.View().Content), "? help", "the keys come back")
}

func TestSaveAllWritesOneFilePerTask(t *testing.T) { // nolint:paralleltest // t.Chdir cannot be used in a parallel test
	dir := t.TempDir()
	t.Chdir(dir)

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 90, Height: 20})
	m = updateTUIModel(t, m, started(1, 0, "build"))
	m = updateTUIModel(t, m, startedUnder(2, 1, 1, "test:unit"))
	m = updateTUIModel(t, m, startedUnder(3, 1, 1, "silent"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 1, name: "build", data: "building\n"})
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "test:unit", data: "testing\n"})

	// Select a task other than the root, to show the folder is named for the
	// run rather than for whatever happens to be selected.
	m.selectedID, m.hasSelect = 2, true

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'S', Text: "S"})
	require.NotNil(t, m.save)
	suggestion := ansi.Strip(m.View().Content)
	assert.Contains(t, suggestion, "Save all to folder:")
	assert.NotContains(t, suggestion, ".log", "saving all asks for a folder, not a file")
	assert.Contains(t, suggestion, "build.", "the folder is named for the task that was run")
	assert.NotContains(t, suggestion, "test-unit")
	m = clearField(t, m)

	m, saved := typePath(t, m, "logs/run-1")
	require.NoError(t, saved.err)
	assert.Equal(t, 2, saved.count, "a task with no output is not written")

	entries, err := os.ReadDir(filepath.Join(dir, "logs", "run-1"))
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	// The folder already says which run this was and when, so a file inside it
	// only says which task it came from. A namespaced task name is not a usable
	// file name.
	assert.ElementsMatch(t, []string{"build.log", "test-unit.log"}, names)
}

func TestSaveReportsWhenThereIsNothingToSave(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "build"))

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	assert.Nil(t, m.save, "nothing to save means nothing to ask about")
	assert.Contains(t, ansi.Strip(m.View().Content), "nothing to save")
}

func TestFileNameForATaskName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "build", fileNameFor("build"))
	assert.Equal(t, "test-unit", fileNameFor("test:unit"))
	assert.Equal(t, "Build-the-docs", fileNameFor("Build the docs"))
	assert.Equal(t, "build-foo", fileNameFor("build:*:foo"))
	// A label can be anything, including nothing usable.
	assert.Equal(t, "task", fileNameFor("///"))
}

func TestGeneratedFileNameGroupsByTask(t *testing.T) {
	t.Parallel()

	name := generatedFileName("2026-09-05T14-30-22", "test:unit")
	assert.Equal(t, "test-unit.2026-09-05T14-30-22.log", name)

	// The task leads, so a folder of logs groups by task and a shell can
	// complete on one without knowing the date. Time ordering is free from
	// ls -t either way.
	older := generatedFileName("2026-09-05T09-00-00", "test:unit")
	other := generatedFileName("2026-09-05T10-00-00", "build")
	sorted := []string{name, older, other}
	slices.Sort(sorted)
	assert.Equal(t, []string{other, older, name}, sorted,
		"a task's logs sort together, oldest first")

	// A timestamp carries no colons, which a file name cannot hold on Windows.
	assert.NotContains(t, name, ":")

	// Two tasks whose names clean up the same way keep separate files.
	used := map[string]bool{}
	first := unusedName(generatedFileName("t", "a:b"), used)
	second := unusedName(generatedFileName("t", "a/b"), used)
	assert.NotEqual(t, first, second)
	assert.True(t, strings.HasSuffix(second, ".log"), second)
}

func TestSaveLeavesAGitignoreInADirectoryItCreated(t *testing.T) { // nolint:paralleltest // t.Chdir cannot be used in a parallel test
	dir := t.TempDir()
	t.Chdir(dir)

	m := withOutput(t)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	_, saved := typePath(t, m, "")
	require.NoError(t, saved.err)

	// Task made the directory, so it ignores itself rather than turning up in
	// git status or being swept in by git add.
	marker, err := os.ReadFile(filepath.Join(dir, "logs", ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(marker), "*")
	assert.Contains(t, string(marker), "Created by Task")
}

func TestSaveLeavesAnExistingDirectoryAlone(t *testing.T) { // nolint:paralleltest // t.Chdir cannot be used in a parallel test
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.Mkdir("logs", 0o750))

	m := withOutput(t)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	_, saved := typePath(t, m, "")
	require.NoError(t, saved.err)

	// The directory was already there, so it is the user's to manage.
	assert.NoFileExists(t, filepath.Join(dir, "logs", ".gitignore"))
}

func TestSaveDoesNotWriteAGitignoreIntoAPathTheUserChose(t *testing.T) { // nolint:paralleltest // t.Chdir cannot be used in a parallel test
	dir := t.TempDir()
	t.Chdir(dir)

	m := withOutput(t)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
	m = clearField(t, m)
	_, saved := typePath(t, m, "build-output/today/build.log")
	require.NoError(t, saved.err)

	assert.NoFileExists(t, filepath.Join(dir, "build-output", ".gitignore"))
	assert.NoFileExists(t, filepath.Join(dir, "build-output", "today", ".gitignore"))
}

func TestFirstPathElement(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "logs", firstPathElement("logs"))
	assert.Equal(t, "logs", firstPathElement("logs/run-1"))
	assert.Equal(t, "logs", firstPathElement("./logs/run-1"))
	assert.Equal(t, ".", firstPathElement("."))
	// An absolute path was asked for by name, so nothing is added to it.
	assert.Empty(t, firstPathElement(filepath.Join(string(filepath.Separator), "var", "logs")))
}

func TestAwaitAnswerGivesUpWhenTheInterfaceStops(t *testing.T) {
	t.Parallel()

	t.Run("an answer is returned", func(t *testing.T) {
		t.Parallel()
		done := make(chan promptAnswer, 1)
		done <- promptAnswer{confirmed: true}
		assert.Equal(t, promptAnswer{confirmed: true}, awaitAnswer(done, make(chan struct{})))
	})

	t.Run("a stopped interface releases the waiting task", func(t *testing.T) {
		t.Parallel()
		// Without this the task waits for an answer that can no longer come,
		// and Task hangs instead of exiting.
		programDone := make(chan struct{})
		close(programDone)
		assert.ErrorIs(t, awaitAnswer(make(chan promptAnswer), programDone).err, task.ErrPromptCancelled)
	})

	t.Run("an answer already given wins a stopped interface", func(t *testing.T) {
		t.Parallel()
		programDone := make(chan struct{})
		close(programDone)

		// The user answered, so their answer is used rather than discarded
		// because the interface happened to stop at the same moment. Repeated
		// because a plain select over two ready cases picks at random.
		for range 100 {
			done := make(chan promptAnswer, 1)
			done <- promptAnswer{value: "dev"}
			answer := awaitAnswer(done, programDone)
			require.NoError(t, answer.err)
			require.Equal(t, "dev", answer.value)
		}
	})
}

func TestPromptingWithoutAnInterfaceDoesNotBlock(t *testing.T) {
	t.Parallel()

	// A task can reach a question after the interface has closed, during a
	// cancelled run. It must be told so rather than waiting for a dialog that
	// will never be drawn.
	ui := &UI{programDone: make(chan struct{})}

	done := make(chan struct{})
	go func() {
		defer close(done)
		confirmed, err := ui.Confirm("deploy", "Really?")
		assert.False(t, confirmed)
		assert.ErrorIs(t, err, task.ErrPromptCancelled)

		value, err := ui.Ask(task.VarRequest{Task: "deploy", Name: "ENV", Type: task.StringVar{}})
		assert.Nil(t, value)
		assert.ErrorIs(t, err, task.ErrPromptCancelled)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("asking without an interface blocked")
	}
}

func TestQuestionsAreAskedOneAtATime(t *testing.T) {
	t.Parallel()

	// Tasks running in parallel can reach questions at once, and the screen
	// holds one. Serialising them is what stops two dialogs racing for it.
	ui := &UI{programDone: make(chan struct{})}

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			_, err := ui.Ask(task.VarRequest{
				Task: "deploy", Name: fmt.Sprintf("VAR_%d", i), Type: task.StringVar{},
			})
			assert.ErrorIs(t, err, task.ErrPromptCancelled)
		})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent questions deadlocked")
	}
}

func TestSendReportsWhetherAnythingReceivedIt(t *testing.T) {
	t.Parallel()

	// Events keep arriving from the executor after the interface has closed;
	// they are dropped rather than panicking on a program that is gone.
	ui := &UI{pending: make(map[uint64]pendingOutput)}
	assert.False(t, ui.send(taskScheduledMsg{}), "nothing is running to receive it")
}

func TestOutputHeaderShowsTheStatusAndExitCode(t *testing.T) {
	t.Parallel()

	exitError := func(name string, status uint8) error {
		return &taskerrors.TaskRunError{TaskName: name, Err: interp.ExitStatus(status)}
	}

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "build"))
	m = updateTUIModel(t, m, started(3, 1, "test"))
	m = updateTUIModel(t, m, started(4, 1, "lint"))

	selectTaskByID(t, &m, 2)
	assert.Contains(t, ansi.Strip(m.outputStatus()), "running", "a task still running says so")

	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	assert.Contains(t, ansi.Strip(m.outputStatus()), "success")
	assert.NotContains(t, m.outputStatus(), "(", "a task that succeeded has no code to report")

	// A task that ran its own failing command reports what it exited with.
	m = updateTUIModel(t, m, taskFinishedMsg{id: 3, result: resultFailed, err: exitError("test", 127)})
	selectTaskByID(t, &m, 3)
	assert.Contains(t, ansi.Strip(m.outputStatus()), "failed (127)")
	assert.Contains(t, ansi.Strip(m.View().Content), "failed (127)")

	// A task that failed because a dependency did carries the dependency's
	// error, which is not this task's exit code.
	m = updateTUIModel(t, m, taskFinishedMsg{id: 4, result: resultFailed, err: exitError("test", 127)})
	selectTaskByID(t, &m, 4)
	assert.Equal(t, "failed", ansi.Strip(m.outputStatus()))
}

func TestOutputHeaderFollowsTheJoinedOwner(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "build"))
	m = updateTUIModel(t, m, scheduled(3, 1, "build"))
	m = updateTUIModel(t, m, taskJoinedMsg{id: 3, ownerID: 2})
	m = updateTUIModel(t, m, taskFinishedMsg{
		id:     2,
		result: resultFailed,
		err:    &taskerrors.TaskRunError{TaskName: "build", Err: interp.ExitStatus(2)},
	})

	selectTaskByID(t, &m, 3)
	assert.Contains(t, ansi.Strip(m.outputStatus()), "failed (2)",
		"a joined invocation reports how the run it waited on ended")
}

func selectTaskByID(t *testing.T, m *tuiModel, id uint64) {
	t.Helper()
	for index, row := range m.taskRows() {
		if row.task.id == id {
			m.selectTask(index)
			require.Equal(t, id, m.selectedID)
			return
		}
	}
	t.Fatalf("no row for task %d", id)
}

func TestScrollbarSitsOnTheOutputPaneBorder(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 70, Height: 14})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "build"))
	selectTaskByID(t, &m, 2)

	assert.NotContains(t, ansi.Strip(m.View().Content), "█",
		"output that fits on screen keeps a plain border")

	lines := make([]string, 0, 40)
	for i := range 40 {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "build", data: strings.Join(lines, "\n")})
	track := m.viewport.Height()

	m.viewport.GotoTop()
	start, size := scrollbarThumb(t, m)
	assert.Equal(t, 0, start, "at the top the thumb starts at the top of the track")
	// Ten lines of forty are on screen, so the thumb covers a quarter of the
	// track.
	assert.Equal(t, track*m.viewport.Height()/40, size)

	m.viewport.GotoBottom()
	start, bottomSize := scrollbarThumb(t, m)
	assert.Equal(t, track-size, start, "at the end the thumb reaches the bottom of the track")
	assert.Equal(t, size, bottomSize, "the thumb keeps its length")

	m.viewport.SetYOffset(15)
	middle, _ := scrollbarThumb(t, m)
	assert.Greater(t, middle, 0)
	assert.Less(t, middle, track-size)
}

// scrollbarThumb reads the thumb's position and length out of the rendered
// output pane, from the border cell at the end of each of its rows.
func scrollbarThumb(t *testing.T, m tuiModel) (start, size int) {
	t.Helper()
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	// The panel opens with its top border and the pane title, and closes with
	// its bottom border; the footer follows it.
	rows := lines[2 : len(lines)-2]
	var thumb []int
	for i, line := range rows {
		runes := []rune(line)
		if runes[len(runes)-1] == '█' {
			thumb = append(thumb, i)
		}
	}
	require.NotEmpty(t, thumb, "no thumb in %q", rows)
	for i, row := range thumb {
		require.Equal(t, thumb[0]+i, row, "the thumb is one unbroken run: %v", thumb)
	}
	return thumb[0], len(thumb)
}

func TestNavigatorKeySwitchesTheTaskView(t *testing.T) {
	t.Parallel()

	prefixFor := func(m tuiModel, id uint64) string {
		for _, row := range m.taskRows() {
			if row.task.id == id {
				return row.treePrefix
			}
		}
		t.Fatalf("no row for task %d", id)
		return ""
	}

	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 90, Height: 14})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, startedUnder(2, 1, 1, "build"))
	m = updateTUIModel(t, m, startedUnder(3, 2, 1, "compile"))

	require.Equal(t, taskNavigatorTree, m.taskNavigator)
	assert.Greater(t, lipgloss.Width(prefixFor(m, 3)), 3, "the tree nests a grandchild under its parent")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'n', Text: "n"})
	assert.Equal(t, taskNavigatorList, m.taskNavigator)
	assert.Equal(t, 3, lipgloss.Width(prefixFor(m, 3)), "the list puts every task under its root")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'n', Text: "n"})
	assert.Equal(t, taskNavigatorTree, m.taskNavigator, "the key toggles back")
}

// fullscreenWith opens the fullscreen output view on a task holding the given
// output, with the cursor on its first line.
func fullscreenWith(t *testing.T, output string, width, height int) tuiModel {
	t.Helper()
	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: width, Height: height})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "build", data: output})
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'f', Text: "f"})
	require.True(t, m.fullscreenOutput)
	m.moveFullscreenCursor(-len(m.fullscreenLines))
	return m
}

func press(t *testing.T, m tuiModel, key rune) tuiModel {
	t.Helper()
	return updateTUIModel(t, m, tea.KeyPressMsg{Code: key, Text: string(key)})
}

func TestFullscreenSelectionCopiesUnwrappedLines(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", 95)
	m := fullscreenWith(t, "first\n"+long+"\nthird\nfourth\n", 60, 12)

	// The long line is folded across two rows, so a line and a row are not the
	// same thing and the cursor has to count lines.
	require.Equal(t, []int{0, 1, 3, 4, 5, 6}, m.fullscreenRowOf)

	m = press(t, m, 'v')
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	require.True(t, m.fullscreenSelecting)

	first, last := m.fullscreenSelectedLines()
	assert.Equal(t, 0, first)
	assert.Equal(t, 2, last, "three lines are selected, not three rows")
	assert.Equal(t, "first\n"+long+"\nthird", m.fullscreenCopyText(false),
		"the copy has the line as it was written, not as it was folded")

	// Every row of every selected line is highlighted, and no row beyond them.
	assert.Equal(t, [2]int{0, 4}, m.fullscreenPainted)
}

func TestFullscreenSelectionGrowsBothWays(t *testing.T) {
	t.Parallel()

	m := fullscreenWith(t, numberedLines(20), 80, 12)
	for range 5 {
		m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	m = press(t, m, 'v')
	for range 2 {
		m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}

	first, last := m.fullscreenSelectedLines()
	assert.Equal(t, 3, first, "moving up from the anchor selects the lines above it")
	assert.Equal(t, 5, last)

	m = press(t, m, 'v')
	assert.False(t, m.fullscreenSelecting, "v again stops extending")
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	first, last = m.fullscreenSelectedLines()
	assert.Equal(t, first, last, "the cursor moves alone once selection has stopped")
}

func TestFullscreenCopyWithoutSelectionTakesEverything(t *testing.T) {
	t.Parallel()

	coloured := "\x1b[31mred\x1b[0m\nplain\n"
	m := fullscreenWith(t, coloured, 80, 12)

	require.False(t, m.fullscreenSelecting)
	assert.Equal(t, "red\nplain\n", m.fullscreenCopyText(false),
		"with nothing selected the key still takes the whole output")
	assert.Equal(t, coloured, m.fullscreenCopyText(true))

	m = press(t, m, 'v')
	assert.Equal(t, "red", m.fullscreenCopyText(false), "y drops the escape sequences")
	assert.Equal(t, "\x1b[31mred\x1b[0m", m.fullscreenCopyText(true), "Y keeps them")
}

func TestFullscreenEscapeClearsTheSelectionBeforeLeaving(t *testing.T) {
	t.Parallel()

	m := fullscreenWith(t, numberedLines(20), 80, 12)
	m = press(t, m, 'v')
	require.True(t, m.fullscreenSelecting)

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.False(t, m.fullscreenSelecting)
	assert.True(t, m.fullscreenOutput, "the first escape clears, it does not leave")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.False(t, m.fullscreenOutput, "the second escape leaves")
}

func TestFullscreenSelectionPinsTheView(t *testing.T) {
	t.Parallel()

	m := fullscreenWith(t, numberedLines(60), 80, 12)
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'G', Text: "G"})
	require.True(t, m.fullscreenViewport.AtBottom())

	m = press(t, m, 'v')
	offset := m.fullscreenViewport.YOffset()
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "build", data: "later\n"})
	assert.Equal(t, offset, m.fullscreenViewport.YOffset(),
		"new output must not drag the view away from lines being picked out")
}

func TestFullscreenSelectionClearedWhenOutputIsTrimmed(t *testing.T) {
	t.Parallel()

	m := fullscreenWith(t, numberedLines(40), 80, 12)
	m = press(t, m, 'v')
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	require.True(t, m.fullscreenSelecting)

	// Output is capped, and trimming it from the front renumbers every line the
	// cursor and the anchor were holding.
	m.byID[2].output = numberedLines(5)
	m.syncFullscreenOutput()

	assert.False(t, m.fullscreenSelecting, "a selection cannot survive its lines being renumbered")
	assert.Less(t, m.fullscreenCursor, len(m.fullscreenLines))
}

func TestFullscreenSelectionAcceptsBothVisualKeys(t *testing.T) {
	t.Parallel()

	for _, key := range []rune{'v', 'V'} {
		m := fullscreenWith(t, numberedLines(20), 80, 12)
		m = press(t, m, key)
		assert.True(t, m.fullscreenSelecting, "%q starts a selection", string(key))
	}
}
