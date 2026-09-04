package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/x/ansi"
)

func (m tuiModel) View() tea.View {
	content := m.renderContent()
	switch {
	case m.showHelp:
		content = m.helpView()
	case m.fullscreenOutput:
		content = m.fullscreenOutputView()
	}
	view := tea.NewView(content)
	view.AltScreen = true
	if m.fullscreenOutput || m.showHelp {
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

	keys := newDashboardKeys(m.focus == outputPane, m.canReturnToLauncher)
	footer := shortHelp(m.help, keys.ShortHelp(), layout.width)
	switch {
	case m.quitting && !m.done:
		footer = renderStatus(layout.width, "stopping tasks… waiting for processes to exit", tuiHelpStyle)
	case m.returning && !m.done:
		footer = renderStatus(layout.width, "stopping tasks… returning to launcher after processes exit", tuiHelpStyle)
	case m.notice != "":
		footer = renderStatus(layout.width, m.notice, tuiTitleStyle)
	}

	return body + "\n" + footer
}

func (m *tuiModel) enterFullscreenOutput() {
	m.fullscreenOutput = true
	view := viewport.New(
		viewport.WithWidth(max(m.width, 1)),
		viewport.WithHeight(max(m.height-1, 1)),
	)
	view.SoftWrap = true
	m.fullscreenViewport = view
	m.fullscreenViewport.SetContent(m.fullscreenOutputContent())
	if m.viewport.AtBottom() {
		m.fullscreenViewport.GotoBottom()
	} else if !m.viewport.AtTop() {
		position := m.viewport.ScrollPercent()
		m.fullscreenViewport.GotoBottom()
		m.fullscreenViewport.SetYOffset(int(position * float64(m.fullscreenViewport.YOffset())))
	}
}

func (m *tuiModel) leaveFullscreenOutput() {
	atTop := m.fullscreenViewport.AtTop()
	atBottom := m.fullscreenViewport.AtBottom()
	position := m.fullscreenViewport.ScrollPercent()
	m.fullscreenOutput = false
	m.fullscreenViewport = viewport.Model{}
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

func (m *tuiModel) syncFullscreenOutput() {
	atBottom := m.fullscreenViewport.AtBottom()
	offset := m.fullscreenViewport.YOffset()
	m.fullscreenViewport.SetContent(m.fullscreenOutputContent())
	if atBottom {
		m.fullscreenViewport.GotoBottom()
	} else {
		m.fullscreenViewport.SetYOffset(offset)
	}
}

func (m *tuiModel) fullscreenOutputContent() string {
	content := ""
	if task := m.selectedTask(); task != nil {
		content = task.output
		if task.truncated {
			content = "… earlier output was discarded …\n" + content
		}
	}
	return content
}

func (m tuiModel) fullscreenOutputView() string {
	footer := renderStatus(m.width, m.notice, tuiTitleStyle)
	if m.notice == "" {
		footer = shortHelp(m.help, newFullscreenKeys().ShortHelp(), m.width)
	}
	return m.fullscreenViewport.View() + "\n" + footer
}

// helpView lists every binding of the view it was opened from. It takes the
// whole screen rather than growing the footer, which would resize the panes.
func (m tuiModel) helpView() string {
	bindings := newDashboardKeys(m.focus == outputPane, m.canReturnToLauncher).allBindings()
	title := "KEYS"
	if m.fullscreenOutput {
		bindings = newFullscreenKeys().allBindings()
		title = "KEYS · fullscreen"
	}
	inner := max(m.width-tuiPanelStyle.GetHorizontalFrameSize(), 1)

	body := tuiPanelStyle.
		BorderForeground(tuiAccentColor).
		Width(max(m.width, 1)).
		Height(max(m.height-1, 1)).
		MaxWidth(max(m.width, 1)).
		MaxHeight(max(m.height-1, 1)).
		Render(paneTitle(title, "", inner) + "\n\n" + fullHelp(m.help, bindings, inner))
	return body + "\n" + renderStatus(m.width, "press any key to return", tuiHelpStyle)
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
	lines := []string{paneTitle("TASKS", m.runStateLabel(), width)}
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

		// The duration is right-aligned so durations line up and can be compared
		// down the column. It is dropped rather than squeezing the name on a
		// narrow pane.
		duration := formatDuration(m.elapsed(row.task))
		available := width - lipgloss.Width(plainPrefix)
		durationWidth := 0
		if duration != "" && available-lipgloss.Width(duration)-1 >= minTaskNameWidth {
			durationWidth = lipgloss.Width(duration) + 1
		} else {
			duration = ""
		}
		withDuration := func(content string, dim bool) string {
			if duration == "" {
				return content
			}
			rendered := duration
			if dim {
				rendered = tuiHelpStyle.Render(duration)
			}
			pad := max(width-lipgloss.Width(content)-lipgloss.Width(duration), 1)
			return content + strings.Repeat(" ", pad) + rendered
		}

		name, status := taskNameStatus(m.taskName(row.task), state, available-durationWidth, m.statusLabels)
		if selected {
			suffix := ""
			if status != "" {
				suffix = " " + status
			}
			lines = append(lines, tuiSelectedStyle.Width(width).Render(withDuration(plainPrefix+name+suffix, false)))
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
			lines = append(lines, withDuration(prefix+tuiRootStyle.Render(name)+suffix, true))
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
		lines = append(lines, withDuration(line, true))
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
	return paneTitle(title, tuiHelpStyle.Render(position), width) + "\n" + m.viewport.View()
}

// paneTitle renders a pane header. right is rendered as given, so a caller can
// style it to carry meaning; the width maths uses its display width.
func paneTitle(left, right string, width int) string {
	left = truncateText(left, max(width-lipgloss.Width(right)-1, 1))
	space := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	return tuiTitleStyle.Render(left) + strings.Repeat(" ", space) + right
}

// runStateLabel summarises the whole run for the task pane header, so the
// footer can stay dedicated to keys.
func (m tuiModel) runStateLabel() string {
	switch {
	case (m.quitting || m.returning) && !m.done:
		return tuiHelpStyle.Render("stopping…")
	case m.done && m.err != nil:
		return tuiFailureStyle.Render("failed")
	case m.done:
		return tuiSuccessStyle.Render("complete")
	case len(m.tasks) == 0:
		return ""
	default:
		return tuiRunningStyle.Render("running")
	}
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

// minTaskNameWidth is the room a name needs before a duration may take space
// from it. Below that, knowing which task a row is matters more than knowing
// how long it took.
const minTaskNameWidth = 12

// formatDuration renders how long a task ran, short enough for a narrow pane.
// Anything under a second is noise in a task runner, so it is left out.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return ""
	case d < 10*time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
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

// appendOutputText appends data to existing, giving a carriage return the
// meaning it has on a terminal: move back to the start of the current line, so
// that what follows redraws it. Progress bars from tools like docker, npm and
// curl repaint themselves that way, and treating every repaint as a new line
// buries the pane in near-identical lines.
//
// A carriage return does not erase anything by itself -- the text stays on
// screen until something overwrites it -- so the pending redraw is carried
// across writes in pendingRedraw and applied only when more output arrives.
//
// The line is replaced rather than overwritten cell by cell, so a repaint
// shorter than what it replaces leaves no remainder behind. That differs from a
// real terminal, but tools that repaint a line pad it to a fixed width, and full
// cursor emulation is well beyond what an output pane needs.
func appendOutputText(existing, data string, pendingRedraw bool) (string, bool) {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	if !pendingRedraw && !strings.ContainsRune(data, '\r') {
		return existing + data, false
	}
	out := existing
	for {
		segment, rest, hasCarriageReturn := strings.Cut(data, "\r")
		out, pendingRedraw = writeOutputSegment(out, segment, pendingRedraw)
		if !hasCarriageReturn {
			return out, pendingRedraw
		}
		pendingRedraw = true
		data = rest
	}
}

// writeOutputSegment appends text containing no carriage return, first dropping
// the line the cursor was returned to if anything is about to redraw it.
func writeOutputSegment(out, segment string, pendingRedraw bool) (string, bool) {
	if segment == "" {
		return out, pendingRedraw
	}
	if pendingRedraw {
		// A newline commits the line the cursor returned to; printable text
		// redraws it.
		if segment[0] != '\n' {
			out = dropCurrentLine(out)
		}
	}
	return out + segment, false
}

func dropCurrentLine(s string) string {
	if newline := strings.LastIndexByte(s, '\n'); newline >= 0 {
		return s[:newline+1]
	}
	return ""
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

// shortHelp renders one line of key hints, clamped to width.
//
// Clamping is ours to do. The help bubble is asked to fit the width but does
// not guarantee it: when there is no room even for its ellipsis it keeps
// appending items, overflowing the screen. It drops from the end, which is why
// the keymaps put help and quit first.
func shortHelp(helpModel help.Model, bindings []key.Binding, width int) string {
	width = max(width, 1)
	// Leave room for the bubble's own ellipsis. Given the exact width it has no
	// space to place one, and rather than stop it keeps appending, leaving us to
	// cut a word in half. With the margin it drops whole entries instead.
	helpModel.SetWidth(max(width-2, 1))
	return truncateText(helpModel.ShortHelpView(bindings), width)
}

// fullHelp renders the key list in as many columns as the width allows, down to
// a single column. Descriptions are written to be read, not to fit four columns
// on an 80 column terminal.
func fullHelp(helpModel help.Model, bindings []key.Binding, width int) string {
	helpModel.ShowAll = true
	for _, columns := range []int{3, 2, 1} {
		view := helpModel.FullHelpView(fullHelpColumns(bindings, columns))
		if lipgloss.Width(view) <= width {
			return view
		}
	}
	return helpModel.FullHelpView(fullHelpColumns(bindings, 1))
}

// newHelpModel styles the help bubble with the palette the rest of the TUI
// uses. Its own defaults are a flat grey that reads as disabled next to the
// panes.
func newHelpModel() help.Model {
	model := help.New()
	styles := model.Styles
	styles.ShortKey, styles.FullKey = tuiKeyStyle, tuiKeyStyle
	styles.ShortDesc, styles.FullDesc = tuiHelpStyle, tuiHelpStyle
	styles.ShortSeparator, styles.FullSeparator = tuiTreeStyle, tuiTreeStyle
	styles.Ellipsis = tuiTreeStyle
	model.Styles = styles
	return model
}

func renderStatus(width int, status string, style lipgloss.Style) string {
	return truncateText(" "+style.Render(status), max(width, 1))
}

var (
	tuiAccentColor       = compat.AdaptiveColor{Light: lipgloss.Color("#006A83"), Dark: lipgloss.Color("#5FD7FF")}
	tuiFilterActiveColor = compat.AdaptiveColor{Light: lipgloss.Color("#5F3DC4"), Dark: lipgloss.Color("#AF87FF")}
	tuiHelpColor         = compat.AdaptiveColor{Light: lipgloss.Color("#66717C"), Dark: lipgloss.Color("#89949F")}
	tuiPanelStyle        = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(compat.AdaptiveColor{Light: lipgloss.Color("#87909A"), Dark: lipgloss.Color("#59636E")}).
				PaddingLeft(1).
				PaddingRight(1)

	tuiTitleStyle        = lipgloss.NewStyle().Bold(true).Foreground(tuiAccentColor)
	tuiFilterActiveStyle = lipgloss.NewStyle().Foreground(tuiFilterActiveColor)
	tuiKeyStyle          = lipgloss.NewStyle().Bold(true).Foreground(tuiAccentColor)
	tuiRootStyle         = lipgloss.NewStyle().Bold(true)
	tuiSelectedStyle     = lipgloss.NewStyle().
				Bold(true).
				Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#10212B"), Dark: lipgloss.Color("#F4F7FA")}).
				Background(compat.AdaptiveColor{Light: lipgloss.Color("#D9E8ED"), Dark: lipgloss.Color("#34444D")})
	tuiTreeStyle     = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#77818A"), Dark: lipgloss.Color("#697580")})
	tuiRunningStyle  = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#8A6500"), Dark: lipgloss.Color("#FFD75F")})
	tuiSuccessStyle  = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#257A3E"), Dark: lipgloss.Color("#5FD787")})
	tuiFailureStyle  = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#B42318"), Dark: lipgloss.Color("#FF6B6B")})
	tuiCanceledStyle = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#5F6670"), Dark: lipgloss.Color("#AAB2BD")})
	tuiHelpStyle     = lipgloss.NewStyle().Foreground(tuiHelpColor)
)
