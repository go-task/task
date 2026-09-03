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
	name    string
	text    string
	running bool
	width   int
	height  int

	stdin  io.Reader
	stdout io.Writer
}

func (s *snapshotOutput) SetStdin(r io.Reader)  { s.stdin = r }
func (s *snapshotOutput) SetStdout(w io.Writer) { s.stdout = w }
func (*snapshotOutput) SetStderr(io.Writer)     {}

func (s *snapshotOutput) Run() error {
	if _, err := io.WriteString(s.stdout, s.blankScreen()+s.body()); err != nil {
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

// blankScreen scrolls whatever the terminal was showing up into its scrollback
// and puts the cursor back at the top, so the snapshot starts on a clean screen
// rather than underneath the shell session.
//
// Erasing the screen instead would also blank it, but terminals disagree on
// whether the erased lines are kept in scrollback or discarded, and discarding
// them would throw away what the user had on screen before Task ran. Scrolling
// destroys nothing on any terminal.
func (s *snapshotOutput) blankScreen() string {
	if s.height <= 0 {
		return ""
	}
	const cursorHome = "\x1b[H"
	return strings.Repeat("\n", s.height) + cursorHome
}

func (s *snapshotOutput) body() string {
	var b strings.Builder
	b.WriteString(s.rule(fmt.Sprintf("snapshot: %s", s.name)))
	b.WriteString("\n")

	if s.text == "" {
		b.WriteString("(no output)\n")
	} else {
		b.WriteString(s.text)
		if !strings.HasSuffix(s.text, "\n") {
			b.WriteString("\n")
		}
	}

	b.WriteString(s.rule("end of snapshot"))
	b.WriteString("\n")
	if s.running {
		// Reuse the colour the navigator gives a running task, which doubles as
		// the usual warning colour: the output above is incomplete.
		b.WriteString(tuiRunningStyle.Render(
			"This task was still running. The output above is what it had produced so far.",
		))
		b.WriteString("\n")
	}
	b.WriteString("Press Enter to return.\n")
	return b.String()
}

// rule draws a labelled horizontal separator, so the snapshot is visibly
// bounded rather than blending into the surrounding terminal output.
func (s *snapshotOutput) rule(label string) string {
	line := "── " + label + " "
	if width := s.width - len([]rune(line)); width > 0 {
		return line + strings.Repeat("─", width)
	}
	return line
}

// snapshotSelectedOutput hands the selected task's output to the terminal.
func (m *tuiModel) snapshotSelectedOutput() tea.Cmd {
	task := m.selectedTask()
	if task == nil {
		return nil
	}
	snapshot := &snapshotOutput{
		name:    m.taskName(task),
		text:    task.output,
		running: m.taskState(task) == taskRunning,
		width:   m.width,
		height:  m.height,
	}
	return tea.Exec(snapshot, func(err error) tea.Msg {
		if err != nil {
			return noticeRequestedMsg{text: fmt.Sprintf("snapshot failed: %v", err)}
		}
		return nil
	})
}
