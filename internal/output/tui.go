package output

import (
	"context"
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
)

const (
	systemTaskName   = "Task messages"
	maxTaskOutputLen = 10 << 20
)

type TUI struct {
	logger *logger.Logger
	input  io.Reader
	output io.Writer

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

func NewTUI(log *logger.Logger) (*TUI, error) {
	if !log.AssumeTerm && !term.IsTerminal() {
		return nil, fmt.Errorf(`task: output style "tui" requires an interactive terminal`)
	}
	return &TUI{
		logger:  log,
		input:   log.Stdin,
		output:  log.Stdout,
		pending: make(map[uint64]pendingOutput),
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

func (t *TUI) TaskStarted(task TaskInvocation) {
	t.send(taskStartedMsg{task: task})
}

func (t *TUI) TaskFinished(id uint64, err error) {
	t.send(taskFinishedMsg{id: id, err: err})
}

func (t *TUI) Run(ctx context.Context, run func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := newTUIModel(cancel)
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
)

type paneFocus uint8

const (
	taskPane paneFocus = iota
	outputPane
)

type tuiTask struct {
	id        uint64
	parentID  uint64
	name      string
	output    string
	state     taskState
	truncated bool

	scrollOffset int
	followOutput bool
}

type taskStartedMsg struct{ task TaskInvocation }
type taskFinishedMsg struct {
	id  uint64
	err error
}
type taskOutputMsg struct {
	id         uint64
	name, data string
}
type outputReadyMsg struct{ tui *TUI }
type executionDoneMsg struct{ err error }

type tuiModel struct {
	tasks      []*tuiTask
	byID       map[uint64]*tuiTask
	selectedID uint64
	hasSelect  bool
	listTop    int
	focus      paneFocus
	width      int
	height     int
	viewport   viewport.Model
	done       bool
	err        error
	cancel     context.CancelFunc
}

func newTUIModel(cancel context.CancelFunc) tuiModel {
	view := viewport.New()
	view.SoftWrap = true
	view.MouseWheelDelta = 3
	return tuiModel{
		byID:     make(map[uint64]*tuiTask),
		width:    100,
		height:   30,
		viewport: view,
		cancel:   cancel,
	}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.saveViewport()
		m.width, m.height = msg.Width, msg.Height
		m.resizeViewport()
		m.loadViewport()
		m.keepSelectionVisible()
		return m, nil
	case taskStartedMsg:
		task := m.ensureTask(msg.task.ID, msg.task.Name, msg.task.ParentID)
		task.state = taskRunning
		m.keepSelectionVisible()
		return m, nil
	case taskFinishedMsg:
		task := m.ensureTask(msg.id, "", 0)
		if msg.err != nil {
			task.state = taskFailed
		} else {
			task.state = taskSucceeded
		}
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
		m.handleMouseClick(tea.Mouse(msg))
		return m, nil
	case tea.MouseWheelMsg:
		return m, m.handleMouseWheel(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
			m.selectTask(0)
		} else {
			m.viewport.GotoTop()
			m.saveViewport()
		}
		return *m, nil
	case "end", "G":
		if m.focus == taskPane {
			m.selectTask(len(m.taskRows()) - 1)
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
	layout := newTUILayout(m.width, m.height)
	left, right := m.renderPanes(layout)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", layout.gap), right)

	help := " tab/←/→ pane  •  ↑/↓ select  •  click a task  •  q quit"
	if m.focus == outputPane {
		help = " tab/←/→ pane  •  ↑/↓ or pgup/pgdn scroll  •  mouse wheel  •  q quit"
	}
	helpStyle := tuiHelpStyle
	if m.done {
		if m.err != nil {
			help = " execution failed  •  enter/q quit"
			helpStyle = tuiFailureStyle
		} else {
			help = " execution complete  •  enter/q quit"
			helpStyle = tuiSuccessStyle
		}
	}
	help = truncateRunes(help, max(layout.width, 1))

	view := tea.NewView(body + "\n" + helpStyle.Render(help))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "Task"
	return view
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
	gap := 1
	leftOuterWidth := min(max(width*30/100, 20), 36)
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
		leftInnerWidth:  max(leftOuterWidth-4, 1),
		rightInnerWidth: max(rightOuterWidth-4, 1),
		innerHeight:     max(bodyHeight-2, 1),
	}
}

func (m *tuiModel) ensureTask(id uint64, name string, parentID uint64) *tuiTask {
	if task, ok := m.byID[id]; ok {
		if name != "" {
			task.name = name
		}
		if parentID != 0 && parentID != id {
			task.parentID = parentID
		}
		return task
	}
	if name == "" {
		name = fmt.Sprintf("task %d", id)
	}
	task := &tuiTask{
		id:           id,
		parentID:     parentID,
		name:         name,
		state:        taskLog,
		followOutput: true,
	}
	m.byID[id] = task
	m.tasks = append(m.tasks, task)
	if !m.hasSelect {
		m.selectedID = id
		m.hasSelect = true
		m.loadViewport()
	}
	return task
}

func (m *tuiModel) appendOutput(id uint64, name, data string) {
	task := m.ensureTask(id, name, 0)
	if task.state == taskLog && id != 0 {
		task.state = taskRunning
	}
	task.output += normalizeOutput(data)
	if len(task.output) > maxTaskOutputLen {
		task.output = task.output[len(task.output)-maxTaskOutputLen:]
		task.truncated = true
	}
	if m.hasSelect && id == m.selectedID {
		m.loadViewport()
	}
}

type tuiTaskRow struct {
	task       *tuiTask
	depth      int
	treePrefix string
}

func (m tuiModel) taskRows() []tuiTaskRow {
	children := make(map[uint64][]*tuiTask)
	var roots []*tuiTask
	for _, task := range m.tasks {
		_, parentExists := m.byID[task.parentID]
		if task.parentID == 0 || task.parentID == task.id || !parentExists {
			roots = append(roots, task)
			continue
		}
		children[task.parentID] = append(children[task.parentID], task)
	}

	rows := make([]tuiTaskRow, 0, len(m.tasks))
	visited := make(map[uint64]bool, len(m.tasks))
	var walk func(*tuiTask, int, []bool, string, bool)
	walk = func(task *tuiTask, depth int, ancestorContinues []bool, connector string, hasNextSibling bool) {
		if visited[task.id] {
			return
		}
		visited[task.id] = true
		var prefix strings.Builder
		for _, continues := range ancestorContinues {
			if continues {
				prefix.WriteString("│  ")
			} else {
				prefix.WriteString("   ")
			}
		}
		prefix.WriteString(connector)
		rows = append(rows, tuiTaskRow{task: task, depth: depth, treePrefix: prefix.String()})
		childAncestors := ancestorContinues
		if depth > 0 {
			childAncestors = append(append([]bool(nil), ancestorContinues...), hasNextSibling)
		}
		for i, child := range children[task.id] {
			hasNext := i < len(children[task.id])-1
			childConnector := "└─ "
			if hasNext {
				childConnector = "├─ "
			}
			walk(child, depth+1, childAncestors, childConnector, hasNext)
		}
	}
	for _, root := range roots {
		walk(root, 0, nil, "", false)
	}
	// Defensive fallback for malformed/cyclic parent information.
	for _, task := range m.tasks {
		if !visited[task.id] {
			walk(task, 0, nil, "", false)
		}
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
		index = 0
	}
	m.selectTask(min(max(index+delta, 0), len(rows)-1))
}

func (m *tuiModel) selectTask(index int) {
	rows := m.taskRows()
	if index < 0 || index >= len(rows) {
		return
	}
	m.saveViewport()
	m.selectedID = rows[index].task.id
	m.hasSelect = true
	m.keepSelectionVisible()
	m.loadViewport()
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
		branch := row.treePrefix
		if lipgloss.Width(branch) > max(width/2, 3) {
			branch = "… " + truncateLeft(branch, max(width/2-2, 1))
		}
		selected := row.task.id == m.selectedID
		marker := "  "
		if selected {
			marker = "▌ "
		}
		plainPrefix := marker + branch + taskIconText(row.task.state) + " "
		name := truncateRunes(row.task.name, max(width-lipgloss.Width(plainPrefix), 1))
		if selected {
			lines = append(lines, tuiSelectedStyle.Width(width).Render(plainPrefix+name))
			continue
		}
		line := tuiTreeStyle.Render(marker+branch) + taskIcon(row.task.state) + " " + name
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m tuiModel) outputPanel(width int) string {
	name := ""
	if task := m.selectedTask(); task != nil {
		name = task.name
	}
	position := ""
	if !m.viewport.AtTop() || !m.viewport.AtBottom() {
		position = fmt.Sprintf("%3.0f%%", m.viewport.ScrollPercent()*100)
	}
	return paneTitle("OUTPUT · "+name, position, width) + "\n" + m.viewport.View()
}

func paneTitle(left, right string, width int) string {
	left = truncateRunes(left, max(width-lipgloss.Width(right)-1, 1))
	space := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	return tuiTitleStyle.Render(left) + strings.Repeat(" ", space) + tuiHelpStyle.Render(right)
}

func taskIcon(state taskState) string {
	switch state {
	case taskRunning:
		return tuiRunningStyle.Render(taskIconText(state))
	case taskSucceeded:
		return tuiSuccessStyle.Render(taskIconText(state))
	case taskFailed:
		return tuiFailureStyle.Render(taskIconText(state))
	default:
		return tuiHelpStyle.Render(taskIconText(state))
	}
}

func taskIconText(state taskState) string {
	switch state {
	case taskRunning:
		return "●"
	case taskSucceeded:
		return "✓"
	case taskFailed:
		return "✗"
	default:
		return "·"
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

func truncateLeft(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[len(runes)-width:])
}

var (
	tuiAccentColor = compat.AdaptiveColor{Light: lipgloss.Color("#006A83"), Dark: lipgloss.Color("#5FD7FF")}
	tuiPanelStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(compat.AdaptiveColor{Light: lipgloss.Color("#87909A"), Dark: lipgloss.Color("#59636E")}).
			PaddingLeft(1).
			PaddingRight(1)
	tuiTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(tuiAccentColor)
	tuiSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#10212B"), Dark: lipgloss.Color("#F4F7FA")}).
				Background(compat.AdaptiveColor{Light: lipgloss.Color("#D9E8ED"), Dark: lipgloss.Color("#34444D")})
	tuiTreeStyle    = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#77818A"), Dark: lipgloss.Color("#697580")})
	tuiRunningStyle = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#8A6500"), Dark: lipgloss.Color("#FFD75F")})
	tuiSuccessStyle = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#257A3E"), Dark: lipgloss.Color("#5FD787")})
	tuiFailureStyle = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#B42318"), Dark: lipgloss.Color("#FF6B6B")})
	tuiHelpStyle    = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#66717C"), Dark: lipgloss.Color("#89949F")})
)
