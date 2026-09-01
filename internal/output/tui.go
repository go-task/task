package output

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/internal/term"
	"github.com/go-task/task/v3/taskfile/ast"
)

const (
	systemTaskName   = "Task messages"
	maxTaskOutputLen = 10 << 20
)

type TUI struct {
	logger       *logger.Logger
	input        io.Reader
	output       io.Writer
	hideInternal bool

	mutex   sync.RWMutex
	program *tea.Program

	outputMutex  sync.Mutex
	pending      map[uint64]pendingOutput
	outputQueued bool
}

type pendingOutput struct {
	name string
	data string
}

func NewTUI(log *logger.Logger, options ast.OutputTUI) (*TUI, error) {
	if !log.AssumeTerm && !term.IsTerminal() {
		return nil, fmt.Errorf(`task: output style "tui" requires an interactive terminal`)
	}
	return &TUI{
		logger:       log,
		input:        log.Stdin,
		output:       log.Stdout,
		hideInternal: options.HideInternal,
		pending:      make(map[uint64]pendingOutput),
	}, nil
}

// WrapWriter satisfies Output. Executor calls WrapWriterForTask so output can
// be associated with a specific invocation; other callers use the system log.
func (t *TUI) WrapWriter(_ io.Writer, _ io.Writer, _ string, _ *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	w := &tuiWriter{tui: t, name: systemTaskName}
	return w, w, func(error) error { return nil }
}

func (t *TUI) WrapWriterForTask(_ io.Writer, _ io.Writer, task TaskInvocation, _ *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	w := &tuiWriter{tui: t, id: task.ID, name: task.Name}
	return w, w, func(error) error { return nil }
}

func (t *TUI) TaskScheduled(task TaskInvocation) {
	t.send(taskScheduledMsg{task: task})
}

func (t *TUI) TaskStarted(task TaskInvocation) {
	t.send(taskStartedMsg{task: task})
}

func (t *TUI) TaskFinished(id uint64, err error) {
	t.send(taskFinishedMsg{id: id, err: err})
}

func (t *TUI) TaskJoined(id, ownerID uint64) {
	t.send(taskJoinedMsg{id: id, ownerID: ownerID})
}

func (t *TUI) Run(ctx context.Context, run func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := newTUIModel(cancel, t.hideInternal)
	program := tea.NewProgram(
		model,
		tea.WithInput(t.input),
		tea.WithOutput(t.output),
	)

	t.mutex.Lock()
	t.program = program
	t.mutex.Unlock()

	oldStdout, oldStderr := t.logger.Stdout, t.logger.Stderr
	systemWriter := &tuiWriter{tui: t, name: systemTaskName}
	t.logger.Stdout, t.logger.Stderr = systemWriter, systemWriter
	defer func() {
		t.logger.Stdout, t.logger.Stderr = oldStdout, oldStderr
		t.mutex.Lock()
		t.program = nil
		t.mutex.Unlock()
	}()

	done := make(chan error, 1)
	go func() {
		<-ctx.Done()
		program.Send(tea.Interrupt())
	}()
	go func() {
		err := run(ctx)
		done <- err
		t.send(executionDoneMsg{err: err})
	}()

	_, uiErr := program.Run()
	cancel()
	runErr := <-done
	if uiErr != nil {
		return fmt.Errorf("task: TUI failed: %w", uiErr)
	}
	return runErr
}

func (t *TUI) send(msg tea.Msg) {
	t.mutex.RLock()
	program := t.program
	t.mutex.RUnlock()
	if program != nil {
		program.Send(msg)
	}
}

func (t *TUI) enqueueOutput(id uint64, name, data string) {
	t.outputMutex.Lock()
	pending := t.pending[id]
	pending.name = name
	pending.data += data
	t.pending[id] = pending
	if t.outputQueued {
		t.outputMutex.Unlock()
		return
	}
	t.outputQueued = true
	t.outputMutex.Unlock()

	// Sending asynchronously lets bursts of command output collapse into one
	// model update instead of rebuilding the viewport for every pipe write.
	go t.send(outputReadyMsg{tui: t})
}

func (t *TUI) drainOutput() map[uint64]pendingOutput {
	t.outputMutex.Lock()
	defer t.outputMutex.Unlock()
	output := t.pending
	t.pending = make(map[uint64]pendingOutput)
	t.outputQueued = false
	return output
}

type tuiWriter struct {
	tui  *TUI
	id   uint64
	name string
}

func (w *tuiWriter) Write(p []byte) (int, error) {
	data := string(append([]byte(nil), p...))
	w.tui.enqueueOutput(w.id, w.name, data)
	return len(p), nil
}

type taskState uint8

const (
	taskLog taskState = iota
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

type taskScheduledMsg struct{ task TaskInvocation }
type taskStartedMsg struct{ task TaskInvocation }
type taskFinishedMsg struct {
	id  uint64
	err error
}
type taskJoinedMsg struct {
	id      uint64
	ownerID uint64
}
type taskOutputMsg struct {
	id         uint64
	name, data string
}
type outputReadyMsg struct{ tui *TUI }
type executionDoneMsg struct{ err error }

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

func (m tuiModel) View() tea.View {
	content := m.renderContent()
	if m.selectingText && m.selectionView != "" {
		content = m.selectionView
	}
	view := tea.NewView(content)
	view.AltScreen = true
	if m.selectingText {
		view.MouseMode = tea.MouseModeNone
	} else {
		view.MouseMode = tea.MouseModeCellMotion
	}
	view.WindowTitle = "Task"
	return view
}

func (m tuiModel) renderContent() string {
	layout := newTUILayout(m.width, m.height)
	left, right := m.renderPanes(layout)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", layout.gap), right)

	help := " tab/←/→ pane  •  ↑/↓ select  •  click a task  •  c select text  •  q quit"
	if m.focus == outputPane {
		help = " tab/←/→ pane  •  ↑/↓ or pgup/pgdn scroll  •  wheel  •  c select text  •  q quit"
	}
	helpStyle := tuiHelpStyle
	if m.done {
		if m.err != nil {
			help = " execution failed  •  c select text  •  enter/q quit"
			helpStyle = tuiFailureStyle
		} else {
			help = " execution complete  •  c select text  •  enter/q quit"
			helpStyle = tuiSuccessStyle
		}
	}
	help = truncateRunes(help, max(layout.width, 1))

	return body + "\n" + helpStyle.Render(help)
}

func (m *tuiModel) enterTextSelection() {
	m.selectingText = true
	view := viewport.New(
		viewport.WithWidth(max(m.width, 1)),
		viewport.WithHeight(max(m.height-1, 1)),
	)
	view.SoftWrap = true
	if task := m.selectedTask(); task != nil {
		content := task.output
		if task.truncated {
			content = "… earlier output was discarded …\n" + content
		}
		view.SetContent(content)
		if m.viewport.AtBottom() {
			view.GotoBottom()
		} else if !m.viewport.AtTop() {
			position := m.viewport.ScrollPercent()
			view.GotoBottom()
			view.SetYOffset(int(position * float64(view.YOffset())))
		}
	}
	m.selectionPage = view
	m.refreshSelectionView()
}

func (m *tuiModel) leaveTextSelection() {
	if m.selectionPage.AtTop() {
		m.viewport.GotoTop()
	} else if m.selectionPage.AtBottom() {
		m.viewport.GotoBottom()
	} else {
		position := m.selectionPage.ScrollPercent()
		m.viewport.GotoBottom()
		m.viewport.SetYOffset(int(position * float64(m.viewport.YOffset())))
	}
	m.saveViewport()
	m.selectingText = false
	m.selectionView = ""
	m.selectionPage = viewport.Model{}
}

func (m *tuiModel) refreshSelectionView() {
	help := truncateRunes(" text selection  •  ↑/↓ or pgup/pgdn scroll  •  drag to select  •  c/esc resume", max(m.width, 1))
	m.selectionView = m.selectionPage.View() + "\n" + tuiHelpStyle.Render(help)
}

func (m tuiModel) renderPanes(layout tuiLayout) (string, string) {
	leftStyle, rightStyle := tuiPanelStyle, tuiPanelStyle
	if m.focus == taskPane {
		leftStyle = leftStyle.BorderForeground(tuiAccentColor)
	} else {
		rightStyle = rightStyle.BorderForeground(tuiAccentColor)
	}
	left := leftStyle.Width(layout.leftOuterWidth).Height(layout.bodyHeight).
		Render(m.taskList(layout.leftInnerWidth, layout.innerHeight))
	right := rightStyle.Width(layout.rightOuterWidth).Height(layout.bodyHeight).
		Render(m.outputPanel(layout.rightInnerWidth))
	return left, right
}

type tuiLayout struct {
	width           int
	bodyHeight      int
	gap             int
	leftOuterWidth  int
	rightOuterWidth int
	leftInnerWidth  int
	rightInnerWidth int
	innerHeight     int
}

func newTUILayout(width, height int) tuiLayout {
	width, height = max(width, 1), max(height, 1)
	bodyHeight := max(height-1, 3)
	horizontalFrame := tuiPanelStyle.GetHorizontalFrameSize()
	verticalFrame := tuiPanelStyle.GetVerticalFrameSize()
	gap := 0
	leftOuterWidth := min(max(width*35/100, 22), 72)
	if right := width - gap - leftOuterWidth; right < 16 {
		leftOuterWidth = max(width-gap-16, 8)
	}
	rightOuterWidth := max(width-gap-leftOuterWidth, 8)
	return tuiLayout{
		width:           width,
		bodyHeight:      bodyHeight,
		gap:             gap,
		leftOuterWidth:  leftOuterWidth,
		rightOuterWidth: rightOuterWidth,
		leftInnerWidth:  max(leftOuterWidth-horizontalFrame, 1),
		rightInnerWidth: max(rightOuterWidth-horizontalFrame, 1),
		innerHeight:     max(bodyHeight-verticalFrame, 1),
	}
}

func (m *tuiModel) scheduleTask(invocation TaskInvocation) *tuiTask {
	if task := m.byID[invocation.ID]; task != nil {
		return task
	}
	isRoot := invocation.ID == invocation.RootID
	key := tuiTaskKey{rootID: invocation.RootID, name: invocation.Name, isRoot: isRoot}
	m.nameCounts[key]++
	task := &tuiTask{
		id:           invocation.ID,
		rootID:       invocation.RootID,
		name:         invocation.Name,
		occurrence:   m.nameCounts[key],
		internal:     invocation.Internal,
		isRoot:       isRoot,
		hidden:       m.hideInternal && invocation.Internal && !isRoot,
		state:        taskLog,
		followOutput: true,
	}
	m.byID[invocation.ID] = task
	m.tasks = append(m.tasks, task)
	if m.hasSelect && m.selectedID == task.id {
		m.loadViewport()
	}
	if !task.isRoot && !task.hidden && !m.hasSelect {
		m.selectedID = task.id
		m.hasSelect = true
		m.loadViewport()
	}
	return task
}

func (m *tuiModel) joinTask(id, ownerID uint64) {
	task := m.byID[id]
	if task == nil {
		return
	}
	selected := m.hasSelect && m.selectedID == id
	if selected {
		m.saveViewport()
	}
	delete(m.byID, id)
	for i, candidate := range m.tasks {
		if candidate.id == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			break
		}
	}
	key := tuiTaskKey{rootID: task.rootID, name: task.name, isRoot: task.isRoot}
	if m.nameCounts[key] > 1 {
		m.nameCounts[key]--
	} else {
		delete(m.nameCounts, key)
	}
	if selected {
		m.selectedID = ownerID
		m.hasSelect = ownerID != 0
		m.loadViewport()
	}
	m.keepSelectionVisible()
}

func (m *tuiModel) ensureOutputTask(id uint64, name string) *tuiTask {
	if task := m.byID[id]; task != nil {
		return task
	}
	if name == "" {
		name = fmt.Sprintf("task %d", id)
	}
	task := &tuiTask{id: id, name: name, state: taskLog, followOutput: true}
	m.byID[id] = task
	m.tasks = append(m.tasks, task)
	if !m.hasSelect {
		m.selectedID = task.id
		m.hasSelect = true
		m.loadViewport()
	}
	return task
}

func (m *tuiModel) appendOutput(id uint64, name, data string) {
	task := m.ensureOutputTask(id, name)
	if task.state == taskLog && id != 0 {
		task.state = taskRunning
	}
	task.output += normalizeOutput(data)
	if len(task.output) > maxTaskOutputLen {
		task.output = task.output[len(task.output)-maxTaskOutputLen:]
		task.truncated = true
	}
	if m.hasSelect && task.id == m.selectedID {
		m.loadViewport()
	}
}

func (m tuiModel) taskName(task *tuiTask) string {
	key := tuiTaskKey{rootID: task.rootID, name: task.name, isRoot: task.isRoot}
	if m.nameCounts[key] > 1 {
		return fmt.Sprintf("#%d %s", task.occurrence, task.name)
	}
	return task.name
}

type tuiTaskRow struct {
	task *tuiTask
}

func (m tuiModel) taskRows() []tuiTaskRow {
	childrenByRoot := make(map[uint64][]*tuiTask)
	var roots, standalone []*tuiTask
	for _, task := range m.tasks {
		if task.hidden {
			continue
		}
		if task.isRoot {
			roots = append(roots, task)
			continue
		}
		if task.rootID == 0 {
			standalone = append(standalone, task)
		} else {
			childrenByRoot[task.rootID] = append(childrenByRoot[task.rootID], task)
		}
	}

	rows := make([]tuiTaskRow, 0, len(m.tasks))
	for _, root := range roots {
		rows = append(rows, tuiTaskRow{task: root})
		for _, child := range childrenByRoot[root.id] {
			rows = append(rows, tuiTaskRow{task: child})
		}
	}
	for _, task := range standalone {
		rows = append(rows, tuiTaskRow{task: task})
	}
	return rows
}

func (m *tuiModel) selectedTask() *tuiTask {
	if !m.hasSelect {
		return nil
	}
	return m.byID[m.selectedID]
}

func (m *tuiModel) selectedIndex() int {
	for i, row := range m.taskRows() {
		if row.task.id == m.selectedID {
			return i
		}
	}
	return -1
}

func (m *tuiModel) moveSelection(delta int) {
	rows := m.taskRows()
	if len(rows) == 0 {
		return
	}
	index := m.selectedIndex()
	if index < 0 {
		if delta < 0 {
			m.selectBoundary(true)
		} else {
			m.selectBoundary(false)
		}
		return
	}
	for index += delta; index >= 0 && index < len(rows); index += delta {
		if !rows[index].task.isRoot {
			m.selectTask(index)
			return
		}
	}
}

func (m *tuiModel) selectTask(index int) {
	rows := m.taskRows()
	if index < 0 || index >= len(rows) || rows[index].task.isRoot {
		return
	}
	m.saveViewport()
	m.selectedID = rows[index].task.id
	m.hasSelect = true
	m.keepSelectionVisible()
	m.loadViewport()
}

func (m *tuiModel) selectBoundary(last bool) {
	rows := m.taskRows()
	if last {
		for i := len(rows) - 1; i >= 0; i-- {
			if !rows[i].task.isRoot {
				m.selectTask(i)
				return
			}
		}
		return
	}
	for i, row := range rows {
		if !row.task.isRoot {
			m.selectTask(i)
			return
		}
	}
}

func (m *tuiModel) keepSelectionVisible() {
	index := m.selectedIndex()
	if index < 0 {
		return
	}
	visible := max(newTUILayout(m.width, m.height).innerHeight-1, 1)
	if index < m.listTop {
		m.listTop = index
	} else if index >= m.listTop+visible {
		m.listTop = index - visible + 1
	}
	maxTop := max(len(m.taskRows())-visible, 0)
	m.listTop = min(max(m.listTop, 0), maxTop)
}

func (m *tuiModel) resizeViewport() {
	layout := newTUILayout(m.width, m.height)
	m.viewport.SetWidth(layout.rightInnerWidth)
	m.viewport.SetHeight(max(layout.innerHeight-1, 1))
}

func (m *tuiModel) loadViewport() {
	task := m.selectedTask()
	if task == nil {
		m.viewport.SetContent("")
		return
	}
	content := task.output
	if task.truncated {
		content = tuiHelpStyle.Render("… earlier output was discarded …") + "\n" + content
	}
	m.viewport.SetContent(content)
	if task.followOutput {
		m.viewport.GotoBottom()
	} else {
		m.viewport.SetYOffset(task.scrollOffset)
	}
}

func (m *tuiModel) saveViewport() {
	task := m.selectedTask()
	if task == nil {
		return
	}
	task.scrollOffset = m.viewport.YOffset()
	task.followOutput = m.viewport.AtBottom()
}

func (m *tuiModel) updateViewport(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	m.saveViewport()
	return cmd
}

func (m *tuiModel) toggleFocus() {
	if m.focus == taskPane {
		m.focus = outputPane
	} else {
		m.focus = taskPane
	}
}

func (m *tuiModel) handleMouseClick(mouse tea.Mouse) {
	layout := newTUILayout(m.width, m.height)
	if mouse.Y < 0 || mouse.Y >= layout.bodyHeight {
		return
	}
	if mouse.X >= 0 && mouse.X < layout.leftOuterWidth {
		m.focus = taskPane
		// Border is row 0 and the title is row 1, so tasks begin at row 2.
		row := mouse.Y - 2
		if row >= 0 {
			m.selectTask(m.listTop + row)
		}
		return
	}
	if mouse.X >= layout.leftOuterWidth+layout.gap {
		m.focus = outputPane
	}
}

func (m *tuiModel) handleMouseWheel(msg tea.MouseWheelMsg) tea.Cmd {
	layout := newTUILayout(m.width, m.height)
	if msg.Y < 0 || msg.Y >= layout.bodyHeight {
		return nil
	}
	if msg.X < layout.leftOuterWidth {
		m.focus = taskPane
		switch msg.Button {
		case tea.MouseWheelUp:
			m.moveSelection(-1)
		case tea.MouseWheelDown:
			m.moveSelection(1)
		}
		return nil
	}
	if msg.X >= layout.leftOuterWidth+layout.gap {
		m.focus = outputPane
		return m.updateViewport(msg)
	}
	return nil
}

func (m tuiModel) taskList(width, height int) string {
	lines := []string{paneTitle("TASKS", "", width)}
	rows := m.taskRows()
	if len(rows) == 0 {
		lines = append(lines, tuiHelpStyle.Render("Waiting for tasks…"))
		return strings.Join(lines, "\n")
	}

	end := min(len(rows), m.listTop+max(height-1, 1))
	for i := m.listTop; i < end; i++ {
		row := rows[i]
		if row.task.isRoot {
			prefix := taskIcon(row.task.state) + " "
			name, status := taskNameStatus(m.taskName(row.task), row.task.state, width-lipgloss.Width(prefix))
			suffix := ""
			if status != "" {
				suffix = " " + taskStateLabel(row.task.state, status)
			}
			lines = append(lines, prefix+tuiRootStyle.Render(name)+suffix)
			continue
		}
		selected := row.task.id == m.selectedID
		marker := " "
		if selected {
			marker = "▌"
		}
		plainPrefix := marker + taskIconText(row.task.state) + " "
		name, status := taskNameStatus(m.taskName(row.task), row.task.state, width-lipgloss.Width(plainPrefix))
		if selected {
			suffix := ""
			if status != "" {
				suffix = " " + status
			}
			lines = append(lines, tuiSelectedStyle.Width(width).Render(plainPrefix+name+suffix))
			continue
		}
		suffix := ""
		if status != "" {
			suffix = " " + taskStateLabel(row.task.state, status)
		}
		line := tuiTreeStyle.Render(marker) + taskIcon(row.task.state) + " " + name + suffix
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m tuiModel) outputPanel(width int) string {
	title := "OUTPUT"
	if task := m.selectedTask(); task != nil {
		title += " · " + m.taskName(task)
	}
	position := ""
	if !m.viewport.AtTop() || !m.viewport.AtBottom() {
		position = fmt.Sprintf("%3.0f%%", m.viewport.ScrollPercent()*100)
	}
	return paneTitle(title, position, width) + "\n" + m.viewport.View()
}

func paneTitle(left, right string, width int) string {
	left = truncateRunes(left, max(width-lipgloss.Width(right)-1, 1))
	space := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	return tuiTitleStyle.Render(left) + strings.Repeat(" ", space) + tuiHelpStyle.Render(right)
}

func taskStateStyle(state taskState) lipgloss.Style {
	switch state {
	case taskRunning:
		return tuiRunningStyle
	case taskSucceeded:
		return tuiSuccessStyle
	case taskFailed:
		return tuiFailureStyle
	case taskCanceled:
		return tuiCanceledStyle
	default:
		return tuiHelpStyle
	}
}

func taskIcon(state taskState) string {
	return taskStateStyle(state).Render(taskIconText(state))
}

func taskStateLabel(state taskState, label string) string {
	return taskStateStyle(state).Render(label)
}

func taskNameStatus(name string, state taskState, width int) (string, string) {
	if width <= 1 {
		return truncateMiddleRunes(name, max(width, 1)), ""
	}
	statusWidth := min(lipgloss.Width(taskStateText(state)), max(width-2, 0))
	if statusWidth == 0 {
		return truncateMiddleRunes(name, width), ""
	}
	status := truncateRunes(taskStateText(state), statusWidth)
	name = truncateMiddleRunes(name, max(width-lipgloss.Width(status)-1, 1))
	return name, status
}

func taskIconText(state taskState) string {
	switch state {
	case taskRunning:
		return "●"
	case taskSucceeded:
		return "✓"
	case taskFailed:
		return "✗"
	case taskCanceled:
		return "■"
	default:
		return "·"
	}
}

func taskStateText(state taskState) string {
	switch state {
	case taskRunning:
		return "running"
	case taskSucceeded:
		return "success"
	case taskFailed:
		return "failed"
	case taskCanceled:
		return "canceled"
	default:
		return "pending"
	}
}

func normalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func truncateRunes(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func truncateMiddleRunes(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	left := (width - 1) / 2
	right := width - 1 - left
	return string(runes[:left]) + "…" + string(runes[len(runes)-right:])
}

var (
	tuiAccentColor = compat.AdaptiveColor{Light: lipgloss.Color("#006A83"), Dark: lipgloss.Color("#5FD7FF")}
	tuiPanelStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(compat.AdaptiveColor{Light: lipgloss.Color("#87909A"), Dark: lipgloss.Color("#59636E")}).
			PaddingLeft(1).
			PaddingRight(1)

	tuiTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(tuiAccentColor)
	tuiRootStyle     = lipgloss.NewStyle().Bold(true)
	tuiSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#10212B"), Dark: lipgloss.Color("#F4F7FA")}).
				Background(compat.AdaptiveColor{Light: lipgloss.Color("#D9E8ED"), Dark: lipgloss.Color("#34444D")})
	tuiTreeStyle     = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#77818A"), Dark: lipgloss.Color("#697580")})
	tuiRunningStyle  = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#8A6500"), Dark: lipgloss.Color("#FFD75F")})
	tuiSuccessStyle  = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#257A3E"), Dark: lipgloss.Color("#5FD787")})
	tuiFailureStyle  = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#B42318"), Dark: lipgloss.Color("#FF6B6B")})
	tuiCanceledStyle = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#5F6670"), Dark: lipgloss.Color("#AAB2BD")})
	tuiHelpStyle     = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#66717C"), Dark: lipgloss.Color("#89949F")})
)
