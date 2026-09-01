package output

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

	got, err = BuildFor(&ast.Output{Name: "tui", TUI: ast.OutputTUI{HideInternal: true, Status: "labels"}}, &logger.Logger{AssumeTerm: true})
	require.NoError(t, err)
	assert.True(t, got.(*TUI).hideInternal)
	assert.True(t, got.(*TUI).statusLabels)

	_, err = BuildFor(&ast.Output{Name: "tui", TUI: ast.OutputTUI{Status: "unknown"}}, &logger.Logger{AssumeTerm: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `expected "icons" or "labels"`)
}

func TestTUIModelTracksTasksAndOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "build"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "build", data: "compiling\r\ndone\r"})
	m = updateTUIModel(t, m, started(3, 1, "test"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	m = updateTUIModel(t, m, taskFinishedMsg{id: 3, err: errors.New("failed")})

	require.Len(t, m.tasks, 3)
	assert.Equal(t, taskSucceeded, m.byID[2].state)
	assert.Equal(t, "compiling\ndone\n", m.byID[2].output)
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

	m := newTUIModel(func() {}, false)
	m.statusLabels = true
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, scheduled(2, 1, "pending-task", false))
	m = updateTUIModel(t, m, started(3, 1, "running-task"))
	m = updateTUIModel(t, m, started(4, 1, "successful-task"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 4})
	m = updateTUIModel(t, m, started(5, 1, "failed-task"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 5, err: errors.New("failed")})
	m = updateTUIModel(t, m, started(6, 1, "canceled-task"))
	m = updateTUIModel(t, m, taskFinishedMsg{id: 6, err: fmt.Errorf("wrapped: %w", context.Canceled)})

	assert.Equal(t, taskCanceled, m.byID[6].state)
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

	m := newTUIModel(func() {}, false)
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

	m := newTUIModel(func() {}, false)
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

	m := newTUIModel(func() {}, false)
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
	assert.Equal(t, "first", m.byID[2].output)
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

func TestTUIModelHidesCallsThatJoinAnExistingExecution(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, scheduled(3, 1, "worker", false))
	require.Len(t, m.tasks, 3)
	assert.Contains(t, m.taskList(30, 10), "#1 worker")
	assert.Contains(t, m.taskList(30, 10), "#2 worker")

	m.selectTask(2)
	m = updateTUIModel(t, m, taskJoinedMsg{id: 3, ownerID: 2})

	require.Len(t, m.tasks, 2)
	assert.Nil(t, m.byID[3])
	assert.Equal(t, uint64(2), m.selectedID)
	assert.Equal(t, []string{"root", "worker"}, rowNames(m.taskRows()))
	assert.NotContains(t, m.taskList(30, 10), "#1")
	assert.NotContains(t, m.taskList(30, 10), "#2")
}

func TestTUIModelRenumbersRemainingRepeatedCallsAfterJoin(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, scheduled(3, 1, "worker", false))
	m = updateTUIModel(t, m, started(4, 1, "worker"))
	m = updateTUIModel(t, m, taskJoinedMsg{id: 3, ownerID: 2})

	assert.Equal(t, 1, m.byID[2].occurrence)
	assert.Equal(t, 2, m.byID[4].occurrence)
	assert.Contains(t, m.taskList(30, 10), "#1 worker")
	assert.Contains(t, m.taskList(30, 10), "#2 worker")
	assert.NotContains(t, m.taskList(30, 10), "#3 worker")
}

func TestTUIModelFlattensExecutionsUnderTheirRoot(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(5, 0, "other-root"))
	m = updateTUIModel(t, m, started(2, 1, "child"))
	m = updateTUIModel(t, m, started(3, 1, "grandchild"))
	m = updateTUIModel(t, m, started(4, 1, "second-child"))

	rows := m.taskRows()
	assert.Equal(t, []string{"root", "child", "grandchild", "second-child", "other-root"}, rowNames(rows))
	list := ansi.Strip(m.taskList(30, 10))
	lines := strings.Split(list, "\n")
	require.GreaterOrEqual(t, len(lines), 5)
	assert.True(t, strings.HasPrefix(lines[1], "● root"), lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "├─ ● child"), lines[2])
	assert.True(t, strings.HasPrefix(lines[3], "├─ ● grandchild"), lines[3])
	assert.True(t, strings.HasPrefix(lines[4], "└─ ● second-child"), lines[4])
}

func TestTUIModelShowsPendingTasksAndDoesNotSelectRoot(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
	m = updateTUIModel(t, m, scheduled(1, 1, "root", false))
	assert.False(t, m.hasSelect)
	m = updateTUIModel(t, m, scheduled(2, 1, "child", false))
	assert.Equal(t, taskPending, m.byID[2].state)
	assert.Equal(t, uint64(2), m.selectedID)

	m.selectTask(0)
	assert.Equal(t, uint64(2), m.selectedID)
	m = updateTUIModel(t, m, taskFinishedMsg{id: 2})
	assert.Equal(t, taskSucceeded, m.byID[2].state)
}

func TestTUIModelCanHideInternalTasks(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, true)
	m = updateTUIModel(t, m, scheduled(1, 1, "root", false))
	m = updateTUIModel(t, m, scheduled(2, 1, "visible", false))
	m = updateTUIModel(t, m, scheduled(3, 1, "internal", true))

	assert.Equal(t, []string{"root", "visible"}, rowNames(m.taskRows()))
	assert.Equal(t, 1, m.nameCounts[tuiTaskKey{rootID: 1, name: "internal"}])
}

func TestTUIModelMouseSelectsTasksAndFocusesPanes(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
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

func TestTUIModelTextSelectionModeDisablesMouseAndFreezesView(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 12})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "first\n"})
	assert.Contains(t, m.View().Content, "c select text")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'c', Text: "c"})
	selectionView := m.View()
	assert.True(t, m.selectingText)
	assert.Equal(t, tea.MouseModeNone, selectionView.MouseMode)
	assert.Contains(t, selectionView.Content, "drag to select")
	assert.Contains(t, selectionView.Content, "first")
	assert.NotContains(t, selectionView.Content, "TASKS")
	assert.NotContains(t, selectionView.Content, "╭")

	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: "second\n"})
	assert.Equal(t, selectionView.Content, m.View().Content)
	assert.Equal(t, "first\nsecond\n", m.byID[2].output)

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.False(t, m.selectingText)
	assert.Equal(t, tea.MouseModeCellMotion, m.View().MouseMode)
	assert.Contains(t, m.View().Content, "second")
}

func TestTUIModelTextSelectionModeScrollsWithKeyboard(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 10})
	m = updateTUIModel(t, m, started(1, 0, "root"))
	m = updateTUIModel(t, m, started(2, 1, "worker"))
	m = updateTUIModel(t, m, taskOutputMsg{id: 2, name: "worker", data: numberedLines(60)})
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'c', Text: "c"})

	require.True(t, m.selectionPage.AtBottom())
	bottomOffset := m.selectionPage.YOffset()
	require.Greater(t, bottomOffset, 0)

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.Less(t, m.selectionPage.YOffset(), bottomOffset)
	assert.Contains(t, m.View().Content, "scroll")

	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'g', Text: "g"})
	assert.True(t, m.selectionPage.AtTop())
	m = updateTUIModel(t, m, tea.KeyPressMsg{Code: 'G', Text: "G"})
	assert.True(t, m.selectionPage.AtBottom())
}

func TestTUIModelScrollsAndRemembersEachTaskOutput(t *testing.T) {
	t.Parallel()

	m := newTUIModel(func() {}, false)
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
	m := newTUIModel(cancel, false)
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

func started(id, rootID uint64, name string) taskStartedMsg {
	if rootID == 0 {
		rootID = id
	}
	return taskStartedMsg{task: TaskInvocation{ID: id, RootID: rootID, Name: name}}
}

func scheduled(id, rootID uint64, name string, internal bool) taskScheduledMsg {
	return taskScheduledMsg{task: TaskInvocation{ID: id, RootID: rootID, Name: name, Internal: internal}}
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

func numberedLines(count int) string {
	var output string
	for i := range count {
		output += fmt.Sprintf("line %02d\n", i)
	}
	return output
}
