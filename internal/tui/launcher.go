package tui

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/go-task/task/v3/taskfile/ast"
)

type launchMode uint8

const (
	launchNormally launchMode = iota
	launchInTUI
)

type launchRequest struct {
	name string
	mode launchMode
}

type launcherItem struct {
	name        string
	description string
}

func (i launcherItem) matches(filter string) bool {
	text := strings.ToLower(i.name + " " + i.description)
	return strings.Contains(text, strings.ToLower(filter))
}

type launcherModel struct {
	items    []launcherItem
	filtered []int
	filter   string
	selected int
	top      int
	width    int
	height   int
}

func newLauncherModel(tasks []*ast.Task) launcherModel {
	m := launcherModel{width: 100, height: 30}
	m.items = make([]launcherItem, 0, len(tasks))
	for _, task := range tasks {
		m.items = append(m.items, launcherItem{
			name:        task.Task,
			description: strings.Join(strings.Fields(task.Desc), " "),
		})
	}
	m.applyFilter("")
	return m
}

func (m launcherModel) Init() tea.Cmd { return nil }

func (m launcherModel) Update(msg tea.Msg) (launcherModel, tea.Cmd, *launchRequest) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.keepSelectionVisible()
	case tea.MouseClickMsg:
		if index := m.itemAtY(msg.Y); index >= 0 {
			m.selected = index
			m.keepSelectionVisible()
		}
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.moveSelection(-1)
		case tea.MouseWheelDown:
			m.moveSelection(1)
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			return m, nil, m.request(launchInTUI)
		case "ctrl+r", "alt+enter":
			return m, nil, m.request(launchNormally)
		case "ctrl+c":
			return m, tea.Quit, nil
		case "up":
			m.moveSelection(-1)
		case "down", "tab":
			m.moveSelection(1)
		case "home":
			m.selectBoundary(false)
		case "end":
			m.selectBoundary(true)
		case "backspace":
			runes := []rune(m.filter)
			if len(runes) > 0 {
				m.applyFilter(string(runes[:len(runes)-1]))
			}
		case "ctrl+u":
			m.applyFilter("")
		case "esc":
			m.applyFilter("")
			return m, nil, nil
		default:
			if text := printableText(msg.Text); text != "" {
				m.applyFilter(m.filter + text)
			}
		}
	}
	return m, nil, nil
}

func (m launcherModel) request(mode launchMode) *launchRequest {
	if m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	return &launchRequest{name: m.items[m.filtered[m.selected]].name, mode: mode}
}

func (m *launcherModel) applyFilter(filter string) {
	selectedName := ""
	if request := m.request(launchNormally); request != nil {
		selectedName = request.name
	}
	m.filter = filter
	m.filtered = m.filtered[:0]
	for index, item := range m.items {
		if item.matches(filter) {
			m.filtered = append(m.filtered, index)
		}
	}
	m.selected = 0
	for index, itemIndex := range m.filtered {
		if m.items[itemIndex].name == selectedName {
			m.selected = index
			break
		}
	}
	m.top = min(m.top, max(len(m.filtered)-1, 0))
	m.keepSelectionVisible()
}

func printableText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}

func (m *launcherModel) moveSelection(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.selected = min(max(m.selected+delta, 0), len(m.filtered)-1)
	m.keepSelectionVisible()
}

func (m *launcherModel) selectBoundary(last bool) {
	if len(m.filtered) == 0 {
		return
	}
	m.selected = 0
	if last {
		m.selected = len(m.filtered) - 1
	}
	m.keepSelectionVisible()
}

func (m *launcherModel) keepSelectionVisible() {
	if len(m.filtered) == 0 {
		m.selected, m.top = 0, 0
		return
	}
	m.selected = min(max(m.selected, 0), len(m.filtered)-1)
	m.top = min(max(m.top, 0), m.selected)
	available := m.taskViewportHeight()
	if m.selected >= m.top+available {
		m.top = m.selected - available + 1
	}
}

func (m launcherModel) taskViewportHeight() int {
	layout := newLauncherLayout(m.width, m.height)
	return max(layout.innerHeight-2, 1)
}

func (m launcherModel) itemAtY(y int) int {
	// The outer border occupies row zero; title and filter occupy rows one and
	// two, so cards begin at row three.
	row := y - 3
	if row < 0 || row >= m.taskViewportHeight() {
		return -1
	}
	index := m.top + row
	if index >= len(m.filtered) {
		return -1
	}
	return index
}

func (m launcherModel) View() tea.View {
	layout := newLauncherLayout(m.width, m.height)
	content := m.renderContent(layout)
	panel := tuiPanelStyle.BorderForeground(tuiAccentColor).
		Width(layout.width).
		Height(layout.bodyHeight).
		Render(content)
	help := renderControls(
		layout.width,
		helpControl{key: "↑/↓", action: "navigate"},
		helpControl{key: "enter", action: "run in TUI"},
		helpControl{key: "ctrl+r", action: "run normally"},
		helpControl{key: "esc", action: "clear"},
		helpControl{key: "ctrl+c", action: "quit"},
	)

	view := tea.NewView(panel + "\n" + help)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "Task"
	return view
}

func (m launcherModel) renderContent(layout launcherLayout) string {
	count := fmt.Sprintf("%d/%d", len(m.filtered), len(m.items))
	lines := []string{paneTitle("TASKS", count, layout.innerWidth)}
	filter := "Filter: " + m.filter
	if m.filter == "" {
		filter = "Filter: type to search"
	}
	lines = append(lines, tuiHelpStyle.Render(truncateText(filter, layout.innerWidth)))

	available := max(layout.innerHeight-len(lines), 0)
	nameWidth := m.nameColumnWidth(layout.innerWidth)
	for index := m.top; index < len(m.filtered) && index-m.top < available; index++ {
		item := m.items[m.filtered[index]]
		lines = append(lines, launcherRow(item, layout.innerWidth, nameWidth, index == m.selected))
	}
	if len(m.filtered) == 0 && available > 0 {
		lines = append(lines, tuiHelpStyle.Render("No matching tasks"))
	}
	return strings.Join(lines, "\n")
}

func (m launcherModel) nameColumnWidth(width int) int {
	longest := 1
	for _, itemIndex := range m.filtered {
		longest = max(longest, lipgloss.Width(m.items[itemIndex].name))
	}
	// Prefer complete task names while preserving useful room for descriptions
	// on ordinary terminal widths.
	maxNameWidth := max(width*3/5, 1)
	if width > 3 {
		maxNameWidth = min(maxNameWidth, width-3)
	}
	return min(longest, maxNameWidth)
}

func launcherRow(item launcherItem, width, nameWidth int, selected bool) string {
	width = max(width, 1)
	nameWidth = min(max(nameWidth, 1), width)
	name := truncateMiddle(item.name, nameWidth)
	name += strings.Repeat(" ", max(nameWidth-lipgloss.Width(name), 0))

	gapWidth := min(2, max(width-nameWidth, 0))
	descriptionWidth := max(width-nameWidth-gapWidth, 0)
	description := truncateText(item.description, descriptionWidth)
	description += strings.Repeat(" ", max(descriptionWidth-lipgloss.Width(description), 0))
	content := name + strings.Repeat(" ", gapWidth) + description
	content += strings.Repeat(" ", max(width-lipgloss.Width(content), 0))

	if selected {
		return tuiSelectedStyle.Width(width).Render(content)
	}
	name = lipgloss.NewStyle().Bold(true).Render(name)
	description = tuiHelpStyle.Render(description)
	return name + strings.Repeat(" ", gapWidth) + description
}

type launcherLayout struct {
	width       int
	bodyHeight  int
	innerWidth  int
	innerHeight int
}

func newLauncherLayout(width, height int) launcherLayout {
	width, height = max(width, 1), max(height, 1)
	bodyHeight := max(height-1, 3)
	return launcherLayout{
		width:       width,
		bodyHeight:  bodyHeight,
		innerWidth:  max(width-tuiPanelStyle.GetHorizontalFrameSize(), 1),
		innerHeight: max(bodyHeight-tuiPanelStyle.GetVerticalFrameSize(), 1),
	}
}
