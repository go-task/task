package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
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

// saveSelected writes the selected task's output to a file.
func (m *tuiModel) saveSelected() tea.Cmd {
	task := m.selectedTask()
	if task == nil || task.output == "" {
		return m.showNotice("nothing to save")
	}
	output := savedOutput{name: m.taskName(task), content: task.output}
	return func() tea.Msg {
		path, err := writeOutputFile(".", output)
		return savedMsg{path: path, count: 1, err: err}
	}
}

// saveAll writes every task's output to its own file in a new directory.
func (m *tuiModel) saveAll() tea.Cmd {
	var outputs []savedOutput
	for _, task := range m.tasks {
		if task.output == "" {
			continue
		}
		outputs = append(outputs, savedOutput{name: m.taskName(task), content: task.output})
	}
	if len(outputs) == 0 {
		return m.showNotice("nothing to save")
	}
	return func() tea.Msg {
		dir, err := makeUnusedDir(".", "task-output")
		if err != nil {
			return savedMsg{err: err}
		}
		for _, output := range outputs {
			if _, err := writeOutputFile(dir, output); err != nil {
				return savedMsg{err: err}
			}
		}
		return savedMsg{path: dir, count: len(outputs)}
	}
}

// writeOutputFile writes one task's output, keeping its escape sequences.
//
// A log is written as the command produced it: cat and less -R render the
// colour, and what is not stripped can still be stripped later.
func writeOutputFile(dir string, output savedOutput) (string, error) {
	path, err := unusedPath(dir, fileNameFor(output.name), ".log")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(output.content), 0o600); err != nil {
		return "", err
	}
	return path, nil
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

// unusedPath returns a path that does not exist yet, so saving twice does not
// overwrite the first result.
func unusedPath(dir, name, extension string) (string, error) {
	for attempt := range 100 {
		candidate := filepath.Join(dir, name+extension)
		if attempt > 0 {
			candidate = filepath.Join(dir, fmt.Sprintf("%s-%d%s", name, attempt, extension))
		}
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no unused name for %q", name)
}

// makeUnusedDir creates a directory that did not exist yet.
func makeUnusedDir(parent, name string) (string, error) {
	for attempt := range 100 {
		candidate := filepath.Join(parent, name)
		if attempt > 0 {
			candidate = filepath.Join(parent, fmt.Sprintf("%s-%d", name, attempt))
		}
		// Mkdir rather than MkdirAll: an existing directory must move us on to
		// the next name rather than have us write into it.
		if err := os.Mkdir(candidate, 0o750); err == nil {
			return candidate, nil
		} else if !os.IsExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("no unused name for %q", name)
}
