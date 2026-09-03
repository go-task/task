package tui

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// snapshotOutput prints a task's output to the terminal and waits, so that the
// terminal's own scrollback and text selection apply to it.
//
// Selecting text inside the dashboard does not work: the terminal drops a
// selection whenever the screen is repainted, and scrolling a viewport is a
// repaint. Handing the text to the terminal sidesteps that entirely, at the
// cost of the view being a snapshot rather than a live one.
type snapshotOutput struct {
	text    string
	running bool

	stdin  io.Reader
	stdout io.Writer
}

func (s *snapshotOutput) SetStdin(r io.Reader)  { s.stdin = r }
func (s *snapshotOutput) SetStdout(w io.Writer) { s.stdout = w }
func (*snapshotOutput) SetStderr(io.Writer)     {}

func (s *snapshotOutput) Run() error {
	if _, err := io.WriteString(s.stdout, s.body()); err != nil {
		return err
	}
	if s.stdin == nil {
		return nil
	}
	// The terminal is back in its normal line-buffered mode here, so a plain
	// read blocks until the user presses Enter.
	_, err := bufio.NewReader(s.stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (s *snapshotOutput) body() string {
	var b strings.Builder
	b.WriteString(s.text)
	if s.text != "" && !strings.HasSuffix(s.text, "\n") {
		b.WriteString("\n")
	}
	if s.text == "" {
		b.WriteString("(no output)\n")
	}
	b.WriteString("\n")
	if s.running {
		b.WriteString("— snapshot taken while the task was still running —\n")
	}
	b.WriteString("Scroll and select with your terminal. Press Enter to return.\n")
	return b.String()
}

// snapshotSelectedOutput hands the selected task's output to the terminal.
func (m *tuiModel) snapshotSelectedOutput() tea.Cmd {
	task := m.selectedTask()
	if task == nil {
		return nil
	}
	snapshot := &snapshotOutput{
		text:    task.output,
		running: m.taskState(task) == taskRunning,
	}
	return tea.Exec(snapshot, func(err error) tea.Msg {
		if err != nil {
			return noticeRequestedMsg{text: fmt.Sprintf("snapshot failed: %v", err)}
		}
		return nil
	})
}
