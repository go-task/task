package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	stamp   string // shared by every file of one save, so a run stays together
	input   textinput.Model
}

// savedAtLayout stamps a file name with when it was saved. Colons are not
// usable in a file name on Windows, so the ISO form is spelled with dashes.
const savedAtLayout = "2006-01-02T15-04-05"

// generatedFileName is what a single saved output is called.
func generatedFileName(stamp, taskName string) string {
	return generatedName(stamp, taskName) + ".log"
}

// generatedName is the name of one run: which task, then when. Used for a
// single file and for the folder a whole run is saved into.
//
// The task leads because time ordering is already free from ls -t, while
// nothing but the name groups a task's logs together. It also lets shell
// completion narrow on a task without having to know the date.
func generatedName(stamp, taskName string) string {
	return fileNameFor(taskName) + "." + stamp
}

// defaultSaveDir is where logs go unless the user says otherwise.
//
// A logs directory beside the project, rather than one shared by every project
// in the home directory, where a "build" log from four repositories would be
// indistinguishable. Visible rather than inside .task, which is Task's own and
// gets deleted when checksums go stale.
func defaultSaveDir() string {
	return "logs"
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

	stamp := time.Now().Format(savedAtLayout)
	name := outputs[0].name
	if all {
		name = m.runName()
	}
	suggestion := filepath.Join(defaultSaveDir(), generatedFileName(stamp, name))
	if all {
		// A folder, not a file: the run is the folder, and the tasks are the
		// files inside it.
		suggestion = filepath.Join(defaultSaveDir(), generatedName(stamp, name))
	}
	input := textinput.New()
	input.Prompt = ""
	input.SetValue(suggestion)
	input.Focus()

	m.save = &saveState{all: all, outputs: outputs, stamp: stamp, input: input}
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
			if err := makeSaveDir(filepath.Dir(path)); err != nil {
				return savedMsg{err: err}
			}
			if err := os.WriteFile(path, []byte(state.outputs[0].content), 0o600); err != nil {
				return savedMsg{err: err}
			}
			return savedMsg{path: path, count: 1}
		}

		if err := makeSaveDir(path); err != nil {
			return savedMsg{err: err}
		}
		used := make(map[string]bool, len(state.outputs))
		for _, output := range state.outputs {
			// The folder already says which run this was and when, so a file
			// only has to say which task it came from.
			name := unusedName(fileNameFor(output.name)+".log", used)
			if err := os.WriteFile(filepath.Join(path, name), []byte(output.content), 0o600); err != nil {
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
	base, extension := strings.TrimSuffix(name, ".log"), ".log"
	for attempt := 1; used[candidate]; attempt++ {
		candidate = fmt.Sprintf("%s-%d%s", base, attempt, extension)
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

// runName names the run, for the folder a whole run is saved into. That is the
// task the user asked for, rather than whichever one happens to be selected.
func (m tuiModel) runName() string {
	var root *tuiTask
	for _, task := range m.tasks {
		if task.isRoot && (root == nil || task.id < root.id) {
			root = task
		}
	}
	if root != nil {
		return m.taskName(root)
	}
	if selected := m.selectedTask(); selected != nil {
		return m.taskName(selected)
	}
	return "task"
}

// ignoreMarker is what Task writes into a logs directory it created, so a
// directory the user did not make does not turn up in git status, or get
// committed by a stray git add.
const ignoreMarker = "# Created by Task. Delete this file to track saved output.\n*\n"

// makeSaveDir creates dir and its parents.
//
// When it is Task that creates the default logs directory, it leaves a
// .gitignore behind. A directory the user typed themselves is theirs, and one
// that already exists is left exactly as it is.
func makeSaveDir(dir string) error {
	head := firstPathElement(dir)
	ours := head == defaultSaveDir() && !pathExists(head)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	if !ours {
		return nil
	}
	marker := filepath.Join(head, ".gitignore")
	if pathExists(marker) {
		return nil
	}
	return os.WriteFile(marker, []byte(ignoreMarker), 0o600)
}

// firstPathElement is the leading directory of a relative path, and empty for
// an absolute one, which the user asked for by name.
func firstPathElement(path string) string {
	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) {
		return ""
	}
	if index := strings.IndexRune(cleaned, filepath.Separator); index >= 0 {
		return cleaned[:index]
	}
	return cleaned
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
