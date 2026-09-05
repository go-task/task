package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/go-task/task/v3/errors"
)

type taskState uint8

const (
	taskPending taskState = iota
	taskRunning
	taskSucceeded
	taskFailed
	taskCanceled
	taskSkipped
)

type paneFocus uint8

const (
	taskPane paneFocus = iota
	outputPane
)

type tuiTaskNavigator uint8

const (
	taskNavigatorList tuiTaskNavigator = iota
	taskNavigatorTree
)

type tuiTask struct {
	id        uint64
	parentID  uint64
	rootID    uint64
	name      string
	isRoot    bool
	shared    bool
	ownerID   uint64
	output    string
	state     taskState
	truncated bool

	// exitCode is what this task's own command exited with, when it failed and
	// reported one.
	exitCode *int

	startedAt  time.Time
	finishedAt time.Time

	scrollOffset int
	followOutput bool

	// pendingRedraw records a carriage return whose line has not been redrawn
	// yet, so the redraw can span separate writes.
	pendingRedraw bool
}

type (
	taskScheduledMsg struct{ task taskInvocation }
	taskStartedMsg   struct {
		task taskInvocation
		at   time.Time
	}
	taskFinishedMsg struct {
		id       uint64
		result   taskResult
		err      error
		at       time.Time
		duration time.Duration
	}
)

type taskJoinedMsg struct {
	id      uint64
	ownerID uint64
}
type taskOutputMsg struct {
	id         uint64
	name, data string
}
type (
	noticeExpiredMsg struct{ id int }
	// elapsedTickMsg redraws running durations. It is only scheduled while a
	// task is running, so a finished dashboard is completely static.
	elapsedTickMsg     struct{}
	noticeRequestedMsg struct{ text string }
)

type (
	outputReadyMsg   struct{ ui *UI }
	executionDoneMsg struct {
		ui  *UI
		err error
	}
	interruptRequestedMsg struct{}
	returnToLauncherMsg   struct{}
)

type tuiModel struct {
	tasks               []*tuiTask
	byID                map[uint64]*tuiTask
	selectedID          uint64
	hasSelect           bool
	listTop             int
	focus               paneFocus
	width               int
	height              int
	viewport            viewport.Model
	done                bool
	quitting            bool
	returning           bool
	err                 error
	cancel              context.CancelFunc
	statusLabels        bool
	taskNavigator       tuiTaskNavigator
	canReturnToLauncher bool

	fullscreenOutput   bool
	fullscreenViewport viewport.Model

	// Fullscreen output is browsed a line at a time. The output is wrapped
	// once into fullscreenRows and handed to the viewport pre-wrapped, so that
	// a viewport row and a screen row are the same thing; fullscreenRowOf maps
	// a logical line to its first row. The cursor and the anchor a selection
	// grows from are logical line indices into fullscreenLines, which is what
	// a copy takes its text from.
	fullscreenLines     []string
	fullscreenRows      []string
	fullscreenShown     []string
	fullscreenRowOf     []int
	fullscreenCursor    int
	fullscreenAnchor    int
	fullscreenSelecting bool
	fullscreenPainted   [2]int
	showHelp            bool
	help                help.Model
	prompt              *promptState
	save                *saveState
	ticking             bool

	// notice is transient feedback shown in place of the controls, such as the
	// result of a copy. noticeID lets a later notice cancel an earlier timer.
	notice   string
	noticeID int
}

type tuiTaskKey struct {
	groupID uint64
	name    string
	isRoot  bool
}

func newTUIModel(cancel context.CancelFunc) tuiModel {
	view := viewport.New()
	view.SoftWrap = true
	view.MouseWheelDelta = 3
	return tuiModel{
		help:          newHelpModel(),
		byID:          make(map[uint64]*tuiTask),
		width:         100,
		height:        30,
		viewport:      view,
		cancel:        cancel,
		taskNavigator: taskNavigatorTree,
	}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.fullscreenOutput {
			m.leaveFullscreenOutput()
		}
		m.saveViewport()
		m.width, m.height = msg.Width, msg.Height
		m.resizeViewport()
		m.loadViewport()
		m.keepSelectionVisible()
		return m, nil
	case taskScheduledMsg:
		m.scheduleTask(msg.task)
		m.keepSelectionVisible()
		return m, nil
	case taskStartedMsg:
		task := m.scheduleTask(msg.task)
		task.state = taskRunning
		// The executor timestamps the event. Stamping it here would measure
		// when this loop got round to it, which for a burst of events is well
		// after the task started.
		task.startedAt = msg.at
		m.keepSelectionVisible()
		return m, m.startElapsedTicker()
	case elapsedTickMsg:
		m.ticking = false
		return m, m.startElapsedTicker()
	case taskFinishedMsg:
		task := m.byID[msg.id]
		if task == nil {
			return m, nil
		}
		task.finishedAt = msg.at
		if msg.duration > 0 {
			// Prefer the executor's measurement over the gap between the two
			// events reaching us.
			task.startedAt = msg.at.Add(-msg.duration)
		}
		switch msg.result {
		case resultSkipped:
			task.state = taskSkipped
		case resultCanceled:
			task.state = taskCanceled
		case resultFailed:
			task.state = taskFailed
			task.exitCode = taskExitCode(task.name, msg.err)
			m.appendFailure(task, msg.err)
		case resultSucceeded:
			task.state = taskSucceeded
		}
		return m, nil
	case noticeExpiredMsg:
		if msg.id == m.noticeID {
			m.notice = ""
		}
		return m, nil
	case noticeRequestedMsg:
		return m, m.showNotice(msg.text)
	case promptRequestedMsg:
		return m, m.beginPrompt(msg.state)
	case savedMsg:
		if msg.err != nil {
			return m, m.showNotice("save failed: " + saveError(msg.err))
		}
		if msg.count == 1 {
			return m, m.showNotice("saved " + msg.path)
		}
		return m, m.showNotice(fmt.Sprintf("saved %d outputs to %s", msg.count, msg.path))
	case clipboardCopiedMsg:
		notice := "copied " + humanizeBytes(msg.size)
		if msg.colours {
			notice += " with colours"
		}
		if !msg.confirmed {
			// Only OSC 52 was sent, and it has no reply, so we cannot know
			// whether the terminal honoured it. Say so rather than claim
			// success: VTE-based terminals silently discard it.
			notice += " — if nothing was copied, press s to save it"
		}
		return m, m.showNotice(notice)
	case taskJoinedMsg:
		m.joinTask(msg.id, msg.ownerID)
		return m, nil
	case taskOutputMsg:
		m.appendOutput(msg.id, msg.name, msg.data)
		return m, nil
	case outputReadyMsg:
		for id, pending := range msg.ui.drainOutput() {
			m.appendOutput(id, pending.name, pending.data)
		}
		return m, nil
	case executionDoneMsg:
		if msg.ui != nil {
			for id, pending := range msg.ui.drainOutput() {
				m.appendOutput(id, pending.name, pending.data)
			}
		}
		for _, task := range m.tasks {
			if task.id == 0 {
				continue
			}
			switch task.state {
			case taskPending:
				task.state = taskSkipped
			case taskRunning:
				task.state = taskCanceled
				task.finishedAt = time.Now()
			}
		}
		m.done, m.err = true, msg.err
		m.reportUnattributedFailure(msg.err)
		if m.quitting {
			return m, tea.Quit
		}
		if m.returning {
			return m, returnToLauncher
		}
		return m, nil
	case interruptRequestedMsg:
		if m.done {
			return m, tea.Quit
		}
		m.quitting = true
		m.cancel()
		return m, nil
	case tea.MouseClickMsg:
		if m.fullscreenOutput {
			return m, nil
		}
		m.handleMouseClick(tea.Mouse(msg))
		return m, nil
	case tea.MouseWheelMsg:
		if m.fullscreenOutput {
			return m, nil
		}
		return m, m.handleMouseWheel(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// reportUnattributedFailure shows a failure that belongs to no task.
//
// A run can fail before anything is scheduled: a declined prompt, a Taskfile
// that will not load. The error is then attached to nothing, and the dashboard
// would say the run failed while showing an empty task list and no reason.
func (m *tuiModel) reportUnattributedFailure(err error) {
	if err == nil {
		return
	}
	for _, task := range m.tasks {
		if task.state == taskFailed {
			// Already shown against the task it belongs to.
			return
		}
	}
	// appendOutput creates the pane and selects it when nothing else is.
	m.appendOutput(0, systemTaskName, err.Error()+"\n")
}

func (m *tuiModel) appendFailure(task *tuiTask, err error) {
	if task.output != "" && !strings.HasSuffix(task.output, "\n") {
		task.output += "\n"
	}
	message := err.Error() + "\n"
	if !strings.HasSuffix(task.output, message) {
		task.output += message
	}
	if selected := m.selectedTask(); selected != nil && selected.id == task.id {
		m.refreshOutputView()
	}
}

func (m *tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.prompt != nil {
		// A task is blocked waiting for this answer, so nothing else can act.
		return m.handlePromptKey(msg)
	}
	if m.save != nil {
		// The footer is a path field; every key belongs to it.
		return m.handleSaveKey(msg)
	}
	if m.showHelp {
		// Any key leaves the key list; it is a reference, not a mode.
		m.showHelp = false
		return *m, nil
	}
	if m.fullscreenOutput {
		return m.handleFullscreenKey(msg)
	}
	return m.handleDashboardKey(msg)
}

func (m *tuiModel) handleFullscreenKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keys := newFullscreenKeys(m.fullscreenSelecting)
	switch {
	case key.Matches(msg, keys.Cancel):
		m.clearFullscreenSelection()
	case key.Matches(msg, keys.Return):
		m.leaveFullscreenOutput()
	case key.Matches(msg, keys.Quit):
		return *m, m.requestQuit()
	case key.Matches(msg, keys.Help):
		m.showHelp = true
	case key.Matches(msg, keys.Select):
		m.toggleFullscreenSelection()
	case key.Matches(msg, keys.Copy):
		return *m, m.copyFullscreenLines(false)
	case key.Matches(msg, keys.CopyRaw):
		return *m, m.copyFullscreenLines(true)
	case key.Matches(msg, keys.Save):
		return *m, m.askWhereToSave(false)
	case key.Matches(msg, keys.SaveAll):
		return *m, m.askWhereToSave(true)
	case key.Matches(msg, keys.Move):
		if msg.String() == "up" || msg.String() == "k" {
			m.moveFullscreenCursor(-1)
		} else {
			m.moveFullscreenCursor(1)
		}
	case key.Matches(msg, keys.Page):
		page := max(m.fullscreenViewport.Height(), 1)
		if msg.String() == "pgup" {
			m.moveFullscreenCursor(-page)
		} else {
			m.moveFullscreenCursor(page)
		}
	case key.Matches(msg, keys.Top):
		m.moveFullscreenCursor(-len(m.fullscreenLines))
	case key.Matches(msg, keys.Bottom):
		m.moveFullscreenCursor(len(m.fullscreenLines))
	}
	return *m, nil
}

// moveFullscreenCursor moves the line cursor and brings it into view, taking
// the selection with it when one is being extended.
func (m *tuiModel) moveFullscreenCursor(delta int) {
	if len(m.fullscreenLines) == 0 {
		return
	}
	m.fullscreenCursor = min(max(m.fullscreenCursor+delta, 0), len(m.fullscreenLines)-1)
	m.keepFullscreenCursorVisible()
	m.paintFullscreenSelection()
}

// keepFullscreenCursorVisible scrolls by as little as it takes to show every
// row of the cursor's line, or its first rows when the line is taller than the
// screen.
func (m *tuiModel) keepFullscreenCursorVisible() {
	first := m.fullscreenRowOf[m.fullscreenCursor]
	last := m.fullscreenRowOf[m.fullscreenCursor+1] - 1
	height := max(m.fullscreenViewport.Height(), 1)
	offset := m.fullscreenViewport.YOffset()
	switch {
	case first < offset:
		offset = first
	case last >= offset+height:
		offset = max(last-height+1, first)
	}
	m.fullscreenViewport.SetYOffset(offset)
}

// toggleFullscreenSelection starts a selection at the cursor, or ends one that
// is already growing. The lines stay selected either way: stopping fixes the
// range so that the cursor can be moved without changing it.
func (m *tuiModel) toggleFullscreenSelection() {
	if len(m.fullscreenLines) == 0 {
		return
	}
	if m.fullscreenSelecting {
		m.fullscreenSelecting = false
	} else {
		m.fullscreenAnchor = m.fullscreenCursor
		m.fullscreenSelecting = true
	}
	m.paintFullscreenSelection()
}

func (m *tuiModel) clearFullscreenSelection() {
	m.fullscreenSelecting = false
	m.fullscreenAnchor = m.fullscreenCursor
	m.paintFullscreenSelection()
}

// fullscreenCopyText is what a copy puts on the clipboard: the selected lines
// as they were written rather than as they were folded to the screen, or the
// whole output when nothing is selected, so that the key keeps the meaning it
// has on the dashboard.
func (m tuiModel) fullscreenCopyText(keepColours bool) string {
	if !m.fullscreenSelecting {
		task := m.selectedTask()
		if task == nil {
			return ""
		}
		return copyText(task.output, keepColours)
	}
	if len(m.fullscreenLines) == 0 {
		return ""
	}
	first, last := m.fullscreenSelectedLines()
	return copyText(strings.Join(m.fullscreenLines[first:last+1], "\n"), keepColours)
}

func (m *tuiModel) copyFullscreenLines(keepColours bool) tea.Cmd {
	text := m.fullscreenCopyText(keepColours)
	if text == "" {
		return m.showNotice("nothing to copy")
	}
	return tea.Batch(
		tea.SetClipboard(text),
		copyToSystemClipboard(text, keepColours),
	)
}

func (m *tuiModel) handleDashboardKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keys := newDashboardKeys(m.focus == outputPane, m.canReturnToLauncher)
	switch {
	case key.Matches(msg, keys.Quit):
		return *m, m.requestQuit()
	case key.Matches(msg, keys.Launcher):
		if m.done {
			return *m, returnToLauncher
		}
		m.returning = true
		m.cancel()
		return *m, nil
	case key.Matches(msg, keys.Help):
		m.showHelp = true
	case key.Matches(msg, keys.Pane):
		m.togglePane(msg)
	case key.Matches(msg, keys.Fullscreen):
		m.enterFullscreenOutput()
	case key.Matches(msg, keys.Copy):
		return *m, m.copyOutput(false)
	case key.Matches(msg, keys.CopyRaw):
		return *m, m.copyOutput(true)
	case key.Matches(msg, keys.Save):
		return *m, m.askWhereToSave(false)
	case key.Matches(msg, keys.SaveAll):
		return *m, m.askWhereToSave(true)
	case key.Matches(msg, keys.Navigator):
		m.toggleTaskNavigator()
	case key.Matches(msg, keys.Page):
		m.focus = outputPane
		return *m, m.updateViewport(msg)
	case key.Matches(msg, keys.Move):
		if m.focus == taskPane {
			if msg.String() == "up" || msg.String() == "k" {
				m.moveSelection(-1)
			} else {
				m.moveSelection(1)
			}
			return *m, nil
		}
		return *m, m.updateViewport(msg)
	case key.Matches(msg, keys.Top):
		if m.focus == taskPane {
			m.selectBoundary(false)
		} else {
			m.viewport.GotoTop()
			m.saveViewport()
		}
	case key.Matches(msg, keys.Bottom):
		if m.focus == taskPane {
			m.selectBoundary(true)
		} else {
			m.viewport.GotoBottom()
			m.saveViewport()
		}
	}
	return *m, nil
}

// togglePane moves focus. The arrow and vi keys name a direction, so they pick
// a pane outright; tab cycles.
func (m *tuiModel) togglePane(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "left", "h":
		m.focus = taskPane
	case "right", "l":
		m.focus = outputPane
	default:
		m.toggleFocus()
	}
}

// requestQuit closes the TUI, cancelling execution first if it is still running.
func (m *tuiModel) requestQuit() tea.Cmd {
	if m.done {
		return tea.Quit
	}
	m.quitting = true
	m.cancel()
	return nil
}

// startElapsedTicker schedules a redraw a second from now, but only while a
// task is running and only if one is not already pending. A dashboard whose
// tasks have all finished draws nothing and costs nothing.
func (m *tuiModel) startElapsedTicker() tea.Cmd {
	if m.ticking || !m.anyRunning() {
		return nil
	}
	m.ticking = true
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return elapsedTickMsg{} })
}

func (m tuiModel) anyRunning() bool {
	for _, task := range m.tasks {
		if task.state == taskRunning {
			return true
		}
	}
	return false
}

// elapsed is how long a task ran, or has been running so far.
func (m tuiModel) elapsed(task *tuiTask) time.Duration {
	if task.startedAt.IsZero() {
		return 0
	}
	if task.finishedAt.IsZero() {
		return time.Since(task.startedAt)
	}
	return task.finishedAt.Sub(task.startedAt)
}

// taskExitCode is the status the task's own command exited with. A task that
// failed because a dependency did carries that dependency's error, so the code
// is reported only when the error names this task.
func taskExitCode(name string, err error) *int {
	runErr, ok := errors.AsType[*errors.TaskRunError](err)
	if !ok || runErr.TaskName != name {
		return nil
	}
	code := runErr.TaskExitCode()
	return &code
}

// toggleTaskNavigator switches the task pane between the tree and the flat
// list. A deep tree is sometimes easier to read flattened, and the choice is
// cheap enough to make while a run is going.
func (m *tuiModel) toggleTaskNavigator() {
	if m.taskNavigator == taskNavigatorTree {
		m.taskNavigator = taskNavigatorList
	} else {
		m.taskNavigator = taskNavigatorTree
	}
	m.keepSelectionVisible()
}

func returnToLauncher() tea.Msg {
	return returnToLauncherMsg{}
}
