package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/x/ansi"
)

func (m tuiModel) View() tea.View {
	content := m.renderContent()
	if m.selectingText {
		content = m.textSelectionView()
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

	controls := []helpControl{
		{key: "tab/←/→", action: "pane"},
		{key: "↑/↓", action: "select"},
		{key: "click", action: "task"},
		{key: "c", action: "copy"},
	}
	if m.focus == outputPane {
		controls = []helpControl{
			{key: "tab/←/→", action: "pane"},
			{key: "↑/↓ or pgup/pgdn", action: "scroll"},
			{key: "wheel", action: "scroll"},
			{key: "c", action: "copy"},
		}
	}
	if m.canReturnToLauncher {
		controls = append(controls, helpControl{key: "esc/b", action: "launcher"})
	}
	controls = append(controls, helpControl{key: "q", action: "quit"})
	help := renderControls(layout.width, controls...)
	if m.quitting && !m.done {
		help = renderStatus(layout.width, "stopping tasks… waiting for processes to exit", tuiHelpStyle)
	} else if m.returning && !m.done {
		help = renderStatus(layout.width, "stopping tasks… returning to launcher after processes exit", tuiHelpStyle)
	} else if m.done {
		doneControls := []helpControl{
			{key: "c", action: "copy"},
			{key: "enter/q", action: "quit"},
		}
		if m.canReturnToLauncher {
			doneControls = append(doneControls, helpControl{key: "esc/b", action: "launcher"})
		}
		if m.err != nil {
			help = renderStatusControls(layout.width, "execution failed", tuiFailureStyle, doneControls...)
		} else {
			help = renderStatusControls(layout.width, "execution complete", tuiSuccessStyle, doneControls...)
		}
	}

	return body + "\n" + help
}

func (m *tuiModel) enterTextSelection() {
	m.selectingText = true
	view := viewport.New(
		viewport.WithWidth(max(m.width, 1)),
		viewport.WithHeight(max(m.height-1, 1)),
	)
	view.SoftWrap = true
	m.selectionPage = view
	m.selectionPage.SetContent(m.textSelectionContent())
	if m.viewport.AtBottom() {
		m.selectionPage.GotoBottom()
	} else if !m.viewport.AtTop() {
		position := m.viewport.ScrollPercent()
		m.selectionPage.GotoBottom()
		m.selectionPage.SetYOffset(int(position * float64(m.selectionPage.YOffset())))
	}
}

func (m *tuiModel) leaveTextSelection() {
	atTop := m.selectionPage.AtTop()
	atBottom := m.selectionPage.AtBottom()
	position := m.selectionPage.ScrollPercent()
	m.selectingText = false
	m.selectionPage = viewport.Model{}
	m.loadViewport()
	if atTop {
		m.viewport.GotoTop()
	} else if atBottom {
		m.viewport.GotoBottom()
	} else {
		m.viewport.GotoBottom()
		m.viewport.SetYOffset(int(position * float64(m.viewport.YOffset())))
	}
	m.saveViewport()
}

func (m *tuiModel) syncSelectionPage() {
	atBottom := m.selectionPage.AtBottom()
	offset := m.selectionPage.YOffset()
	m.selectionPage.SetContent(m.textSelectionContent())
	if atBottom {
		m.selectionPage.GotoBottom()
	} else {
		m.selectionPage.SetYOffset(offset)
	}
}

func (m *tuiModel) textSelectionContent() string {
	content := ""
	if task := m.selectedTask(); task != nil {
		content = task.output
		if task.truncated {
			content = "… earlier output was discarded …\n" + content
		}
	}
	return content
}

func (m tuiModel) textSelectionView() string {
	help := renderStatusControls(
		m.width,
		"text selection",
		tuiHelpStyle,
		helpControl{key: "↑/↓ or pgup/pgdn", action: "scroll"},
		helpControl{key: "drag", action: "select"},
		helpControl{key: "c/esc", action: "resume"},
	)
	return m.selectionPage.View() + "\n" + help
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
		state := m.taskState(row.task)
		selected := row.task.id == m.selectedID
		sharedPrefix := ""
		if m.taskNavigator == taskNavigatorTree && row.task.shared {
			sharedPrefix = "↳ "
		}
		plainIcon := ""
		if !m.statusLabels {
			plainIcon = taskIconText(state) + " "
		}
		plainPrefix := row.treePrefix + plainIcon + sharedPrefix
		name, status := taskNameStatus(m.taskName(row.task), state, width-lipgloss.Width(plainPrefix), m.statusLabels)
		if selected {
			suffix := ""
			if status != "" {
				suffix = " " + status
			}
			lines = append(lines, tuiSelectedStyle.Width(width).Render(plainPrefix+name+suffix))
			continue
		}
		if row.task.isRoot {
			prefix := ""
			if !m.statusLabels {
				prefix = taskIcon(state) + " "
			}
			prefix += sharedPrefix
			suffix := ""
			if status != "" {
				suffix = " " + taskStateLabel(state, status)
			}
			lines = append(lines, prefix+tuiRootStyle.Render(name)+suffix)
			continue
		}
		suffix := ""
		if status != "" {
			suffix = " " + taskStateLabel(state, status)
		}
		icon := ""
		if !m.statusLabels {
			icon = taskIcon(state) + " "
		}
		line := tuiTreeStyle.Render(row.treePrefix) + icon + tuiTreeStyle.Render(sharedPrefix) + name + suffix
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m tuiModel) outputPanel(width int) string {
	title := "OUTPUT"
	if task := m.selectedRowTask(); task != nil {
		title += " · " + m.taskName(task)
	}
	position := ""
	if !m.viewport.AtTop() || !m.viewport.AtBottom() {
		position = fmt.Sprintf("%3.0f%%", m.viewport.ScrollPercent()*100)
	}
	return paneTitle(title, position, width) + "\n" + m.viewport.View()
}

func paneTitle(left, right string, width int) string {
	left = truncateText(left, max(width-lipgloss.Width(right)-1, 1))
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
	case taskSkipped:
		return tuiHelpStyle
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

func taskNameStatus(name string, state taskState, width int, showStatus bool) (string, string) {
	if !showStatus {
		return truncateMiddle(name, max(width, 1)), ""
	}
	if width <= 1 {
		return truncateMiddle(name, max(width, 1)), ""
	}
	statusWidth := min(lipgloss.Width(taskStateText(state)), max(width-2, 0))
	if statusWidth == 0 {
		return truncateMiddle(name, width), ""
	}
	status := truncateText(taskStateText(state), statusWidth)
	name = truncateMiddle(name, max(width-lipgloss.Width(status)-1, 1))
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
	case taskSkipped:
		return "○"
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
	case taskSkipped:
		return "skipped"
	default:
		return "pending"
	}
}

func normalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func truncateText(s string, width int) string {
	return ansi.Truncate(s, max(width, 0), "…")
}

func truncateMiddle(s string, width int) string {
	stringWidth := ansi.StringWidth(s)
	if stringWidth <= width {
		return s
	}
	if width <= 1 {
		return ansi.Truncate("…", max(width, 0), "")
	}
	left := (width - 1) / 2
	right := width - 1 - left
	return ansi.Cut(s, 0, left) + "…" + ansi.Cut(s, stringWidth-right, stringWidth)
}

type helpControl struct {
	key    string
	action string
}

func renderControls(width int, controls ...helpControl) string {
	return renderStatusControls(width, "", tuiHelpStyle, controls...)
}

func renderStatus(width int, status string, style lipgloss.Style) string {
	return truncateText(" "+style.Render(status), max(width, 1))
}

func renderStatusControls(width int, status string, style lipgloss.Style, controls ...helpControl) string {
	var line strings.Builder
	line.WriteString(" ")
	if status != "" {
		line.WriteString(style.Render(status))
	}
	for index, control := range controls {
		if status != "" || index > 0 {
			line.WriteString("   ")
		}
		line.WriteString(tuiKeyStyle.Render(control.key))
		line.WriteString(tuiHelpStyle.Render(": " + control.action))
	}
	return truncateText(line.String(), max(width, 1))
}

var (
	tuiAccentColor = compat.AdaptiveColor{Light: lipgloss.Color("#006A83"), Dark: lipgloss.Color("#5FD7FF")}
	tuiPanelStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(compat.AdaptiveColor{Light: lipgloss.Color("#87909A"), Dark: lipgloss.Color("#59636E")}).
			PaddingLeft(1).
			PaddingRight(1)

	tuiTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(tuiAccentColor)
	tuiKeyStyle      = lipgloss.NewStyle().Bold(true).Foreground(tuiAccentColor)
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
