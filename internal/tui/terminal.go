package tui

import "io"

// terminalRequestedMsg asks the program to release the terminal so that run can
// use it. Only the update loop can start a handover, but what needs one is the
// executor, on another goroutine, so the request travels as a message and the
// outcome comes back on done.
type terminalRequestedMsg struct {
	ui   *UI
	run  func() error
	done chan error
}

// terminalHandover gives the terminal to run for as long as it takes. Bubble
// Tea leaves the alternate screen and restores the terminal around it, which is
// what lets Task's prompter start a program of its own: nesting one inside ours
// would not work, but this is not nested.
type terminalHandover struct {
	ui  *UI
	run func() error
	err error
}

// borrowedTerminal is where Task's own messages go while it holds the terminal.
type borrowedTerminal struct {
	out io.Writer
}

func (*terminalHandover) SetStdin(io.Reader)  {}
func (*terminalHandover) SetStdout(io.Writer) {}
func (*terminalHandover) SetStderr(io.Writer) {}

// Run always reports success, so that a prompt the user declined is not
// mistaken for a failure to hand over the terminal. The real outcome is read
// from err.
func (h *terminalHandover) Run() error {
	// Task's own messages are normally collected into a pane. While it holds
	// the terminal they belong on the terminal: a confirmation prompt asked
	// into a pane the user cannot see would hang waiting for an answer to a
	// question nobody was shown.
	h.ui.borrowed.Store(&borrowedTerminal{out: h.ui.output})
	defer h.ui.borrowed.Store(nil)

	h.err = h.run()
	return nil
}

// runInTerminal lends the terminal to fn and waits for it to finish.
//
// Serialised, because only one thing can hold the terminal, and because Task
// may reach two prompts at once when tasks run in parallel.
func (t *UI) runInTerminal(fn func() error) error {
	t.terminalMutex.Lock()
	defer t.terminalMutex.Unlock()

	t.mutex.RLock()
	program := t.program
	t.mutex.RUnlock()
	if program == nil {
		// Nothing is drawing, so the terminal is already free.
		return fn()
	}

	done := make(chan error, 1)
	program.Send(terminalRequestedMsg{ui: t, run: fn, done: done})
	select {
	case err := <-done:
		return err
	case <-t.programDone:
		// The program stopped before it could hand anything over. The terminal
		// is free either way, and fn has not run.
		return fn()
	}
}
