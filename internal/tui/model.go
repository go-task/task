package tui

import (
	"context"
	"strings"

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

	scrollOffset int
	followOutput bool

	// pendingRedraw records a carriage return whose line has not been redrawn
	// yet, so the redraw can span separate writes.
	pendingRedraw bool
}

type (
	taskScheduledMsg struct{ task taskInvocation }
	taskStartedMsg   struct{ task taskInvocation }
	taskFinishedMsg  struct {
		id  uint64
		err error
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
	noticeExpiredMsg   struct{ id int }
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
		m.keepSelectionVisible()
		return m, nil
	case taskFinishedMsg:
		task := m.byID[msg.id]
		if task == nil {
			return m, nil
		}
		if errors.Is(msg.err, errTaskSkipped) {
			task.state = taskSkipped
		} else if errors.Is(msg.err, context.Canceled) {
			task.state = taskCanceled
		} else if msg.err != nil {
			task.state = taskFailed
			m.appendFailure(task, msg.err)
		} else {
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
			}
		}
		m.done, m.err = true, msg.err
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
	if m.fullscreenOutput {
		switch msg.String() {
		case "f", "esc":
			m.leaveFullscreenOutput()
			return *m, nil
		case "y":
			return *m, m.copyOutput()
		case "s":
			return *m, m.snapshotSelectedOutput()
		case "q", "ctrl+c":
			if m.done {
				return *m, tea.Quit
			}
			m.quitting = true
			m.cancel()
			return *m, nil
		case "up", "k":
			m.fullscreenViewport.ScrollUp(1)
		case "down", "j":
			m.fullscreenViewport.ScrollDown(1)
		case "pgup":
			m.fullscreenViewport.PageUp()
		case "pgdown":
			m.fullscreenViewport.PageDown()
		case "home", "g":
			m.fullscreenViewport.GotoTop()
		case "end", "G":
			m.fullscreenViewport.GotoBottom()
		default:
			return *m, nil
		}
		return *m, nil
	}
	switch msg.String() {
	case "q", "ctrl+c":
		if m.done {
			return *m, tea.Quit
		}
		m.quitting = true
		m.cancel()
		return *m, nil
	case "b", "esc":
		if !m.canReturnToLauncher {
			if m.done {
				return *m, tea.Quit
			}
			m.quitting = true
			m.cancel()
			return *m, nil
		}
		if m.done {
			return *m, returnToLauncher
		}
		m.returning = true
		m.cancel()
		return *m, nil
	case "tab", "shift+tab":
		m.toggleFocus()
		return *m, nil
	case "left", "h":
		m.focus = taskPane
		return *m, nil
	case "right", "l":
		m.focus = outputPane
		return *m, nil
	case "f":
		m.enterFullscreenOutput()
		return *m, nil
	case "y":
		return *m, m.copyOutput()
	case "s":
		return *m, m.snapshotSelectedOutput()
	case "pgup", "pgdown":
		m.focus = outputPane
		return *m, m.updateViewport(msg)
	case "up", "k":
		if m.focus == taskPane {
			m.moveSelection(-1)
			return *m, nil
		}
		return *m, m.updateViewport(msg)
	case "down", "j":
		if m.focus == taskPane {
			m.moveSelection(1)
			return *m, nil
		}
		return *m, m.updateViewport(msg)
	case "home", "g":
		if m.focus == taskPane {
			m.selectBoundary(false)
		} else {
			m.viewport.GotoTop()
			m.saveViewport()
		}
		return *m, nil
	case "end", "G":
		if m.focus == taskPane {
			m.selectBoundary(true)
		} else {
			m.viewport.GotoBottom()
			m.saveViewport()
		}
		return *m, nil
	case "enter":
		if m.done {
			return *m, tea.Quit
		}
	}
	return *m, nil
}

func returnToLauncher() tea.Msg {
	return returnToLauncherMsg{}
}
