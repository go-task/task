package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/go-task/task/v3"
)

// promptKind is which question is being asked.
type promptKind uint8

const (
	promptConfirm promptKind = iota
	promptText
	promptChoice
)

// promptAnswer travels back to the goroutine that asked.
type promptAnswer struct {
	confirmed bool
	value     any
	err       error
}

// promptState is a question waiting on screen. Only one exists at a time: the
// task that asked is blocked until it is answered.
type promptState struct {
	kind    promptKind
	task    string
	message string
	name    string
	options []string
	cursor  int
	input   textinput.Model
	done    chan promptAnswer
}

type promptRequestedMsg struct{ state *promptState }

// Confirm asks whether to run a task that declares "prompt".
func (t *UI) Confirm(taskName, message string) (bool, error) {
	answer := t.ask(&promptState{kind: promptConfirm, task: taskName, message: message})
	return answer.confirmed, answer.err
}

// Ask asks for a required variable that was not supplied.
func (t *UI) Ask(request task.VarRequest) (any, error) {
	state := &promptState{kind: promptText, task: request.Task, name: request.Name}
	switch varType := request.Type.(type) {
	case task.EnumVar:
		state.kind = promptChoice
		state.options = varType.Options
	case task.StringVar:
	default:
		// Guessing would produce a value the task then acts on.
		return nil, fmt.Errorf("task: the TUI cannot ask for a %T variable", varType)
	}
	answer := t.ask(state)
	return answer.value, answer.err
}

// ask puts a question on screen and waits for it to be answered. Serialised,
// because the screen holds one question at a time and tasks may ask at once.
func (t *UI) ask(state *promptState) promptAnswer {
	t.promptMutex.Lock()
	defer t.promptMutex.Unlock()

	t.mutex.RLock()
	program := t.program
	t.mutex.RUnlock()
	if program == nil {
		return promptAnswer{err: task.ErrPromptCancelled}
	}

	state.done = make(chan promptAnswer, 1)
	program.Send(promptRequestedMsg{state: state})
	select {
	case answer := <-state.done:
		return answer
	case <-t.programDone:
		// The interface stopped before the question could be answered.
		return promptAnswer{err: task.ErrPromptCancelled}
	}
}

// beginPrompt puts a question on screen.
func (m *tuiModel) beginPrompt(state *promptState) tea.Cmd {
	if state.kind == promptText {
		input := textinput.New()
		input.Prompt = ""
		input.Placeholder = "type a value"
		input.Focus()
		state.input = input
	}
	m.prompt = state
	if state.kind == promptText {
		return textinput.Blink
	}
	return nil
}

// answerPrompt hands the answer back and takes the question off screen.
func (m *tuiModel) answerPrompt(answer promptAnswer) {
	if m.prompt == nil {
		return
	}
	m.prompt.done <- answer
	m.prompt = nil
}

func (m *tuiModel) handlePromptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	state := m.prompt
	if msg.String() == "ctrl+c" {
		// Ctrl+C closes the interface, as it does everywhere else. Answer the
		// question first, or the task waiting on it never returns.
		m.answerPrompt(promptAnswer{err: task.ErrPromptCancelled})
		return *m, m.requestQuit()
	}
	switch state.kind {
	case promptConfirm:
		switch msg.String() {
		case "y", "Y":
			m.answerPrompt(promptAnswer{confirmed: true})
		case "n", "N", "esc", "enter":
			m.answerPrompt(promptAnswer{})
		}
	case promptChoice:
		switch msg.String() {
		case "up", "k":
			state.cursor = max(state.cursor-1, 0)
		case "down", "j":
			state.cursor = min(state.cursor+1, len(state.options)-1)
		case "enter":
			if len(state.options) > 0 {
				m.answerPrompt(promptAnswer{value: state.options[state.cursor]})
			}
		case "esc":
			m.answerPrompt(promptAnswer{err: task.ErrPromptCancelled})
		}
	case promptText:
		switch msg.String() {
		case "enter":
			m.answerPrompt(promptAnswer{value: state.input.Value()})
		case "esc":
			m.answerPrompt(promptAnswer{err: task.ErrPromptCancelled})
		default:
			var cmd tea.Cmd
			state.input, cmd = state.input.Update(msg)
			return *m, cmd
		}
	}
	return *m, nil
}

// promptView draws the question as a dialog over the dashboard.
//
// A blocking question is an interruption, not a place you navigated to, and a
// box over the interface reads that way. It also tells it apart from the key
// list, which fills the screen because it is a reference you asked for.
func (m tuiModel) promptView() string {
	width, height := max(m.width, 1), max(m.height, 1)
	box := m.promptBox(width, height)
	x := max((width-lipgloss.Width(box))/2, 0)
	y := max((height-lipgloss.Height(box))/2, 0)

	// A Compositor is what applies a layer's position; Canvas.Compose draws
	// into the whole canvas and ignores it.
	return lipgloss.NewCanvas(width, height).
		Compose(lipgloss.NewCompositor(
			lipgloss.NewLayer(m.renderContent()),
			lipgloss.NewLayer(box).X(x).Y(y).Z(1),
		)).
		Render()
}

// promptBox is the dialog itself, sized to its content within the screen.
func (m tuiModel) promptBox(screenWidth, screenHeight int) string {
	state := m.prompt
	outer := min(max(screenWidth-8, 24), 72)
	inner := max(outer-tuiPanelStyle.GetHorizontalFrameSize(), 1)
	// A long list of options must not grow the box past the screen.
	maxHeight := max(screenHeight-2, 3)

	var body strings.Builder
	body.WriteString(tuiTitleStyle.Render(truncateText(
		fmt.Sprintf("Task %q is asking", state.task), inner)))
	body.WriteString("\n\n")

	switch state.kind {
	case promptConfirm:
		body.WriteString(wrapText(state.message, inner))
	case promptText:
		body.WriteString(wrapText(state.name, inner))
		body.WriteString("\n\n")
		state.input.SetWidth(max(inner-1, 1))
		body.WriteString(state.input.View())
	case promptChoice:
		body.WriteString(wrapText(state.name, inner))
		body.WriteString("\n")
		for i, option := range state.options {
			line := truncateText("  "+option, inner)
			if i == state.cursor {
				// Highlight the whole row, as the launcher does.
				line = tuiSelectedStyle.Width(inner).Render(line)
			}
			body.WriteString("\n" + line)
		}
	}

	return tuiPanelStyle.
		BorderForeground(tuiAccentColor).
		Width(outer).
		MaxWidth(outer).
		MaxHeight(maxHeight).
		Render(body.String())
}

// promptKeys are the footer hints while a question is on screen.
func (m tuiModel) promptKeys() []helpBinding {
	switch m.prompt.kind {
	case promptConfirm:
		return []helpBinding{{"y", "yes"}, {"n/esc", "no"}, {"ctrl+c", "quit"}}
	case promptChoice:
		return []helpBinding{{"↑/↓", "choose"}, {"enter", "confirm"}, {"esc", "cancel"}}
	default:
		return []helpBinding{{"enter", "confirm"}, {"esc", "cancel"}}
	}
}

// wrapText breaks text to width without splitting words, for a message written
// by a Taskfile author who did not know how wide the pane would be.
func wrapText(text string, width int) string {
	width = max(width, 1)
	var out strings.Builder
	line := ""
	for word := range strings.FieldsSeq(text) {
		switch {
		case line == "":
			line = word
		case lipgloss.Width(line)+1+lipgloss.Width(word) <= width:
			line += " " + word
		default:
			out.WriteString(truncateText(line, width) + "\n")
			line = word
		}
	}
	out.WriteString(truncateText(line, width))
	return out.String()
}

type helpBinding struct{ key, desc string }

func renderPromptKeys(m tuiModel, width int, keys []helpBinding) string {
	var line strings.Builder
	for i, k := range keys {
		if i > 0 {
			line.WriteString(m.help.Styles.ShortSeparator.Inline(true).Render(m.help.ShortSeparator))
		}
		line.WriteString(m.help.Styles.ShortKey.Inline(true).Render(k.key))
		line.WriteString(" ")
		line.WriteString(m.help.Styles.ShortDesc.Inline(true).Render(k.desc))
	}
	return truncateText(line.String(), max(width, 1))
}
