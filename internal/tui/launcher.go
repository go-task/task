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

func (i launcherItem) height() int {
	if i.description == "" {
		return 3
	}
	return 4
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
			return m, nil, m.request(launchNormally)
		case "ctrl+t", "alt+enter":
			return m, nil, m.request(launchInTUI)
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
			if m.filter != "" {
				m.applyFilter("")
				return m, nil, nil
			}
			return m, tea.Quit, nil
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
	available := m.cardViewportHeight()
	for m.top < m.selected && m.cardsHeight(m.top, m.selected) > available {
		m.top++
	}
}

func (m launcherModel) cardsHeight(from, through int) int {
	height := 0
	for index := from; index <= through && index < len(m.filtered); index++ {
		height += m.items[m.filtered[index]].height()
	}
	return height
}

func (m launcherModel) cardViewportHeight() int {
	layout := newLauncherLayout(m.width, m.height)
	return max(layout.innerHeight-2, 1)
}

func (m launcherModel) itemAtY(y int) int {
	// The outer border occupies row zero; title and filter occupy rows one and
	// two, so cards begin at row three.
	row := y - 3
	if row < 0 || row >= m.cardViewportHeight() {
		return -1
	}
	for index := m.top; index < len(m.filtered); index++ {
		height := m.items[m.filtered[index]].height()
		if row < height {
			return index
		}
		row -= height
	}
	return -1
}

func (m launcherModel) View() tea.View {
	layout := newLauncherLayout(m.width, m.height)
	content := m.renderContent(layout)
	panel := tuiPanelStyle.BorderForeground(tuiAccentColor).
		Width(layout.width).
		Height(layout.bodyHeight).
		Render(content)
	help := truncateText(" type to filter  •  ↑/↓ select  •  enter run normally  •  ctrl+t run in TUI  •  esc clear  •  ctrl+c quit", layout.width)

	view := tea.NewView(panel + "\n" + tuiHelpStyle.Render(help))
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
	used := 0
	for index := m.top; index < len(m.filtered); index++ {
		item := m.items[m.filtered[index]]
		card := launcherCard(item, layout.innerWidth, index == m.selected)
		if used+len(card) > available {
			break
		}
		lines = append(lines, card...)
		used += len(card)
	}
	if len(m.filtered) == 0 && available > 0 {
		lines = append(lines, tuiHelpStyle.Render("No matching tasks"))
	}
	return strings.Join(lines, "\n")
}

func launcherCard(item launcherItem, width int, selected bool) []string {
	width = max(width, 4)
	borderStyle := tuiTreeStyle
	if selected {
		borderStyle = lipgloss.NewStyle().Foreground(tuiAccentColor).Bold(true)
	}
	top := borderStyle.Render("╭" + strings.Repeat("─", width-2) + "╮")
	bottom := borderStyle.Render("╰" + strings.Repeat("─", width-2) + "╯")
	lines := []string{top, launcherCardLine(item.name, width, selected, true)}
	if item.description != "" {
		lines = append(lines, launcherCardLine(item.description, width, selected, false))
	}
	return append(lines, bottom)
}

func launcherCardLine(text string, width int, selected, bold bool) string {
	innerWidth := max(width-2, 1)
	text = truncateText(text, max(innerWidth-2, 1))
	content := " " + text
	content += strings.Repeat(" ", max(innerWidth-lipgloss.Width(content), 0))
	style := lipgloss.NewStyle()
	if bold {
		style = style.Bold(true)
	}
	if selected {
		style = style.Foreground(tuiSelectedStyle.GetForeground()).Background(tuiSelectedStyle.GetBackground())
	} else if !bold {
		style = tuiHelpStyle
	}
	borderStyle := tuiTreeStyle
	if selected {
		borderStyle = lipgloss.NewStyle().Foreground(tuiAccentColor).Bold(true)
	}
	return borderStyle.Render("│") + style.Render(content) + borderStyle.Render("│")
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
