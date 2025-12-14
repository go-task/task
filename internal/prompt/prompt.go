package prompt

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	return model.options[model.cursor], nil
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
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	return textModel{
		varName:   varName,
		textInput: ti,
	}
}

func (m textModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.value = m.textInput.Value()
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textModel) View() string {
	if m.done {
		return ""
	}

	prompt := promptStyle.Render(fmt.Sprintf("? Enter value for %s: ", m.varName))
	return prompt + m.textInput.View() + "\n"
}

// selectModel is the Bubble Tea model for selection
type selectModel struct {
	varName   string
	options   []string
	cursor    int
	cancelled bool
	done      bool
}

func newSelectModel(varName string, options []string) selectModel {
	return selectModel{
		varName: varName,
		options: options,
		cursor:  0,
	}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case tea.KeyUp, tea.KeyShiftTab:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown, tea.KeyTab:
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		}

		// Also handle j/k for vim users
		switch msg.String() {
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

func (m selectModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString(promptStyle.Render(fmt.Sprintf("? Select value for %s:", m.varName)))
	b.WriteString("\n")

	for i, opt := range m.options {
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("❯ "))
			b.WriteString(selectedStyle.Render(opt))
		} else {
			b.WriteString("  " + opt)
		}
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render("  (↑/↓ to move, enter to select, esc to cancel)"))

	return b.String()
}
