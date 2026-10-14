package input

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/go-task/task/v3/errors"
)

var ErrCancelled = errors.New("prompt cancelled")

var (
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true) // cyan bold
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true) // cyan bold
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true) // green bold
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))            // gray
)

// Prompter handles interactive variable prompting
type Prompter struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Text prompts the user for a text value
func (p *Prompter) Text(varName string) (string, error) {
	m := newTextModel(varName)

	prog := tea.NewProgram(m,
		tea.WithInput(p.Stdin),
		tea.WithOutput(p.Stderr),
	)

	result, err := prog.Run()
	if err != nil {
		return "", err
	}

	model := result.(textModel)
	if model.cancelled {
		return "", ErrCancelled
	}

	return model.value, nil
}

// Select prompts the user to select from a list of options
func (p *Prompter) Select(varName string, options []string) (string, error) {
	if len(options) == 0 {
		return "", errors.New("no options provided")
	}

	m := newSelectModel(varName, options)

	prog := tea.NewProgram(m,
		tea.WithInput(p.Stdin),
		tea.WithOutput(p.Stderr),
	)

	result, err := prog.Run()
	if err != nil {
		return "", err
	}

	model := result.(selectModel)
	if model.cancelled {
		return "", ErrCancelled
	}

	return model.value, nil
}

// Prompt prompts for a variable value, using Select if enum is provided, Text otherwise
func (p *Prompter) Prompt(varName string, enum []string) (string, error) {
	if len(enum) > 0 {
		return p.Select(varName, enum)
	}
	return p.Text(varName)
}

// textModel is the Bubble Tea model for text input
type textModel struct {
	varName   string
	textInput textinput.Model
	value     string
	cancelled bool
	done      bool
}

func newTextModel(varName string) textModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 256
	ti.SetWidth(40)
	ti.Focus()

	return textModel{
		varName:   varName,
		textInput: ti,
	}
}

func (m textModel) Init() tea.Cmd {
	return tea.Batch(m.textInput.Focus(), textinput.Blink)
}

func (m textModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Keystroke() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			m.value = m.textInput.Value()
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	prompt := promptStyle.Render(fmt.Sprintf("? Enter value for %s: ", m.varName))
	return tea.NewView(prompt + m.textInput.View() + "\n")
}

const maxVisibleOptions = 10

// selectModel is the Bubble Tea model for selection
type selectModel struct {
	varName   string
	options   []string
	matches   []int
	search    textinput.Model
	query     string
	filtering bool
	cursor    int
	offset    int
	pageSize  int
	width     int
	height    int
	value     string
	cancelled bool
	done      bool
}

func newSelectModel(varName string, options []string) selectModel {
	search := textinput.New()
	search.Prompt = "  / "
	search.Placeholder = "Search choices"
	search.CharLimit = 256
	search.SetWidth(76)
	matches := make([]int, len(options))
	for i := range options {
		matches[i] = i
	}
	return selectModel{
		varName:  varName,
		options:  options,
		matches:  matches,
		search:   search,
		pageSize: maxVisibleOptions,
		width:    80,
		height:   24,
	}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// A non-terminal input can report an unknown (zero) size.
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		m.pageSize = max(1, min(maxVisibleOptions, m.height-4))
		m.search.SetWidth(max(1, m.width-4))
	case tea.KeyPressMsg:
		switch msg.Keystroke() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "up", "shift+tab":
			m.cursor--
		case "down", "tab":
			m.cursor++
		case "pgup":
			m.cursor -= m.pageSize
		case "pgdown":
			m.cursor += m.pageSize
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = len(m.matches) - 1
		case "enter":
			if len(m.matches) > 0 {
				m.value = m.options[m.matches[m.cursor]]
				m.done = true
				return m, tea.Quit
			}
		case "/":
			if !m.filtering {
				m.filtering = true
				return m, m.search.Focus()
			}
			m.search, cmd = m.search.Update(msg)
		case "j", "k":
			if m.filtering {
				m.search, cmd = m.search.Update(msg)
			} else if msg.Keystroke() == "j" {
				m.cursor++
			} else {
				m.cursor--
			}
		default:
			m.search, cmd = m.search.Update(msg)
		}
	default:
		m.search, cmd = m.search.Update(msg)
	}

	if query := m.search.Value(); query != m.query {
		// Filter only when the query changes, never on cursor movement.
		m.filterChoices(query)
	}
	m.cursor = max(0, min(m.cursor, len(m.matches)-1))
	m.offset = min(m.offset, max(0, len(m.matches)-m.pageSize))
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+m.pageSize {
		m.offset = m.cursor - m.pageSize + 1
	}
	return m, cmd
}

func (m *selectModel) filterChoices(query string) {
	m.query = query
	m.matches = m.matches[:0]
	query = strings.ToLower(query)
	for i, option := range m.options {
		if strings.Contains(strings.ToLower(option), query) {
			m.matches = append(m.matches, i)
		}
	}
	m.cursor = 0
	m.offset = 0
}

func (m selectModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	var b strings.Builder

	b.WriteString(promptStyle.Render(fmt.Sprintf("? Select value for %s:", m.varName)))
	b.WriteString("\n")
	if m.filtering {
		b.WriteString(m.search.View())
	} else {
		b.WriteString(dimStyle.Render("  / Search choices"))
	}
	b.WriteString("\n")

	end := min(m.offset+m.pageSize, len(m.matches))
	if len(m.matches) == 0 {
		b.WriteString(dimStyle.Render("  No matches. Ctrl+U clears the search."))
		b.WriteString("\n")
	}
	for i := m.offset; i < end; i++ {
		opt := strings.ReplaceAll(strings.ReplaceAll(m.options[m.matches[i]], "\n", " "), "\r", " ")
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("❯ "))
			b.WriteString(selectedStyle.Render(opt))
		} else {
			b.WriteString("  " + opt)
		}
		b.WriteString("\n")
	}

	start := 0
	if len(m.matches) > 0 {
		start = m.offset + 1
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %d–%d of %d choices", start, end, len(m.matches))))
	if m.search.Value() != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf(" (%d total)", len(m.options))))
	}
	b.WriteString("\n")
	help := "  ↑/↓ move · / search · Enter select · Esc cancel"
	if m.width < 50 {
		help = "  ↑/↓ · / search · Enter · Esc"
	}
	b.WriteString(dimStyle.Render(help))

	return tea.NewView(lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(b.String()))
}
