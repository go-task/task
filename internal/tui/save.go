package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/go-task/task/v3/errors"
)

// savedMsg reports the outcome of writing output to disk.
type savedMsg struct {
	path  string
	count int
	err   error
}

// savedOutput is one task's output, copied out of the model so the writing can
// happen off the update loop.
type savedOutput struct {
	name    string
	content string
}

// saveState is a pending save, waiting for the user to say where.
type saveState struct {
	all     bool
	outputs []savedOutput
	input   textinput.Model
}

// askWhereToSave puts a path field in the footer, filled in with a default so
// that Enter alone is enough.
func (m *tuiModel) askWhereToSave(all bool) tea.Cmd {
	var outputs []savedOutput
	if all {
		for _, task := range m.tasks {
			if task.output != "" {
				outputs = append(outputs, savedOutput{name: m.taskName(task), content: task.output})
			}
		}
	} else if task := m.selectedTask(); task != nil && task.output != "" {
		outputs = append(outputs, savedOutput{name: m.taskName(task), content: task.output})
	}
	if len(outputs) == 0 {
		return m.showNotice("nothing to save")
	}

	suggestion := "task-output"
	if !all {
		suggestion = fileNameFor(outputs[0].name) + ".log"
	}
	input := textinput.New()
	input.Prompt = ""
	input.SetValue(suggestion)
	input.Focus()

	m.save = &saveState{all: all, outputs: outputs, input: input}
	return textinput.Blink
}

func (m *tuiModel) handleSaveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		state := m.save
		m.save = nil
		return *m, saveOutputs(state, strings.TrimSpace(state.input.Value()))
	case "esc", "ctrl+c":
		m.save = nil
		return *m, nil
	}
	var cmd tea.Cmd
	m.save.input, cmd = m.save.input.Update(msg)
	return *m, cmd
}

// saveOutputs writes to the path the user gave, creating any directories it
// needs. The path was typed deliberately, so an existing file is replaced, as
// a shell redirect would.
func saveOutputs(state *saveState, target string) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return savedMsg{err: errors.New("no path given")}
		}
		path, err := expandHome(target)
		if err != nil {
			return savedMsg{err: err}
		}

		if !state.all {
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				return savedMsg{err: err}
			}
			if err := os.WriteFile(path, []byte(state.outputs[0].content), 0o600); err != nil {
				return savedMsg{err: err}
			}
			return savedMsg{path: path, count: 1}
		}

		if err := os.MkdirAll(path, 0o750); err != nil {
			return savedMsg{err: err}
		}
		used := make(map[string]bool, len(state.outputs))
		for _, output := range state.outputs {
			name := unusedName(fileNameFor(output.name), used)
			file := filepath.Join(path, name+".log")
			if err := os.WriteFile(file, []byte(output.content), 0o600); err != nil {
				return savedMsg{err: err}
			}
		}
		return savedMsg{path: path, count: len(state.outputs)}
	}
}

// unusedName keeps two tasks whose names clean up to the same thing from
// writing over each other.
func unusedName(name string, used map[string]bool) string {
	candidate := name
	for attempt := 1; used[candidate]; attempt++ {
		candidate = fmt.Sprintf("%s-%d", name, attempt)
	}
	used[candidate] = true
	return candidate
}

// expandHome resolves a leading ~, which a user typing a path will expect.
func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

// fileNameFor makes a task name safe to use as a file name. Task names carry
// namespace colons and wildcards, and a label can be any text at all.
func fileNameFor(name string) string {
	var out strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			out.WriteRune(r)
		default:
			out.WriteRune('-')
		}
	}
	cleaned := strings.Trim(collapseDashes(out.String()), "-.")
	if cleaned == "" {
		return "task"
	}
	return cleaned
}

func collapseDashes(s string) string {
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

// saveFooter renders the path field in place of the key hints.
func (m tuiModel) saveFooter(width int) string {
	label := "Save to: "
	if m.save.all {
		label = "Save all to folder: "
	}
	keys := renderPromptKeys(m, width, []helpBinding{{"enter", "save"}, {"esc", "cancel"}})

	room := max(width-lipgloss.Width(label)-lipgloss.Width(keys)-3, 8)
	input := m.save.input
	input.SetWidth(room)
	line := tuiTitleStyle.Render(label) + input.View() + "  " + keys
	return truncateText(line, max(width, 1))
}

// saveError is what to tell the user when a save fails.
//
// A filesystem error repeats the path, which the user typed a moment ago and
// can still see. Keeping it pushes the reason, the only part they do not know,
// off the end of the footer.
func saveError(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}
	return err.Error()
}
