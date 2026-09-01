package output

import (
	"context"

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
)

type paneFocus uint8

const (
	taskPane paneFocus = iota
	outputPane
)

type tuiTask struct {
	id         uint64
	rootID     uint64
	name       string
	occurrence int
	internal   bool
	isRoot     bool
	hidden     bool
	output     string
	state      taskState
	truncated  bool

	scrollOffset int
	followOutput bool
}

type (
	taskScheduledMsg struct{ task TaskInvocation }
	taskStartedMsg   struct{ task TaskInvocation }
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
	outputReadyMsg   struct{ tui *TUI }
	executionDoneMsg struct{ err error }
)

type tuiModel struct {
	tasks        []*tuiTask
	byID         map[uint64]*tuiTask
	nameCounts   map[tuiTaskKey]int
	selectedID   uint64
	hasSelect    bool
	listTop      int
	focus        paneFocus
	width        int
	height       int
	viewport     viewport.Model
	done         bool
	err          error
	cancel       context.CancelFunc
	hideInternal bool
	statusLabels bool

	selectingText bool
	selectionView string
	selectionPage viewport.Model
}

type tuiTaskKey struct {
	rootID uint64
	name   string
	isRoot bool
}

func newTUIModel(cancel context.CancelFunc, hideInternal bool) tuiModel {
	view := viewport.New()
	view.SoftWrap = true
	view.MouseWheelDelta = 3
	return tuiModel{
		byID:         make(map[uint64]*tuiTask),
		nameCounts:   make(map[tuiTaskKey]int),
		width:        100,
		height:       30,
		viewport:     view,
		cancel:       cancel,
		hideInternal: hideInternal,
	}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.selectingText {
			m.leaveTextSelection()
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
		if errors.Is(msg.err, context.Canceled) {
			task.state = taskCanceled
		} else if msg.err != nil {
			task.state = taskFailed
		} else {
			task.state = taskSucceeded
		}
		return m, nil
	case taskJoinedMsg:
		m.joinTask(msg.id, msg.ownerID)
		return m, nil
	case taskOutputMsg:
		m.appendOutput(msg.id, msg.name, msg.data)
		return m, nil
	case outputReadyMsg:
		for id, pending := range msg.tui.drainOutput() {
			m.appendOutput(id, pending.name, pending.data)
		}
		return m, nil
	case executionDoneMsg:
		m.done, m.err = true, msg.err
		return m, nil
	case tea.InterruptMsg:
		m.cancel()
		return m, tea.Quit
	case tea.MouseClickMsg:
		if m.selectingText {
			return m, nil
		}
		m.handleMouseClick(tea.Mouse(msg))
		return m, nil
	case tea.MouseWheelMsg:
		if m.selectingText {
			return m, nil
		}
		return m, m.handleMouseWheel(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.selectingText {
		switch msg.String() {
		case "c", "esc":
			m.leaveTextSelection()
			return *m, nil
		case "q", "ctrl+c":
			if !m.done {
				m.cancel()
			}
			return *m, tea.Quit
		case "up", "k":
			m.selectionPage.ScrollUp(1)
		case "down", "j":
			m.selectionPage.ScrollDown(1)
		case "pgup":
			m.selectionPage.PageUp()
		case "pgdown":
			m.selectionPage.PageDown()
		case "home", "g":
			m.selectionPage.GotoTop()
		case "end", "G":
			m.selectionPage.GotoBottom()
		default:
			return *m, nil
		}
		m.refreshSelectionView()
		return *m, nil
	}
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		if !m.done {
			m.cancel()
		}
		return *m, tea.Quit
	case "tab", "shift+tab":
		m.toggleFocus()
		return *m, nil
	case "left", "h":
		m.focus = taskPane
		return *m, nil
	case "right", "l":
		m.focus = outputPane
		return *m, nil
	case "c":
		m.enterTextSelection()
		return *m, nil
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
