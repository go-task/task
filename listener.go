package task

import (
	"io"

	"github.com/go-task/task/v3/errors"
)

// Invocation identifies one runtime call to a task. IDs are unique within an
// Executor, including repeated calls to the same task.
type Invocation struct {
	ID       uint64 // Unique call ID
	ParentID uint64 // ID of the task call that scheduled this call; zero for roots
	RootID   uint64 // ID of the root call requested by the user
	Name     string
}

// ErrSkipped is reported to a Listener's TaskFinished when Task decided not to
// run a call at all -- because the task is not for the current platform, or its
// "if" condition was not met. It is never returned to the caller of Run: from
// Task's point of view a skipped call is not a failure.
var ErrSkipped = errors.New("task: skipped")

// Listener observes task execution and may take over the terminal while it
// runs. It is optional: an Executor without one behaves exactly as before.
//
// Lifecycle methods are called from the goroutines that run the tasks, so
// implementations must be safe for concurrent use.
type Listener interface {
	// TaskScheduled reports a call that Task intends to run. A call that is
	// attempted is always reported to TaskFinished, but a scheduled call may
	// never be attempted -- when an earlier requested root fails, say -- and
	// then no further event arrives for it.
	TaskScheduled(Invocation)
	// TaskStarted reports that a call began executing its deps and commands.
	TaskStarted(Invocation)
	// TaskFinished reports the outcome of a call. A nil error means success; an
	// error wrapping context.Canceled means the call was interrupted, and
	// ErrSkipped means Task chose not to run it.
	TaskFinished(id uint64, err error)
	// TaskJoined reports a call that waits on the execution owned by ownerID
	// instead of running its own. It produces no output of its own.
	TaskJoined(id, ownerID uint64)

	// WriterFor returns the destination streams for a call's command output.
	// Returning a nil writer leaves that stream on the Executor's own, which is
	// what a listener that only wants lifecycle events should do.
	WriterFor(Invocation) (stdOut, stdErr io.Writer)

	// OwnsTerminal reports that the listener is drawing to the terminal, so Task
	// must not write to its own streams or run anything interactive.
	OwnsTerminal() bool
}

func (e *Executor) notifyScheduled(invocation Invocation) {
	if e.Listener != nil {
		e.Listener.TaskScheduled(invocation)
	}
}

func (e *Executor) notifyStarted(invocation Invocation) {
	if e.Listener != nil {
		e.Listener.TaskStarted(invocation)
	}
}

func (e *Executor) notifyFinished(id uint64, err error) {
	if e.Listener != nil {
		e.Listener.TaskFinished(id, err)
	}
}

func (e *Executor) notifyJoined(id, ownerID uint64) {
	if e.Listener != nil {
		e.Listener.TaskJoined(id, ownerID)
	}
}

// ownsTerminal reports whether a listener is drawing to the terminal.
func (e *Executor) ownsTerminal() bool {
	return e.Listener != nil && e.Listener.OwnsTerminal()
}

// listenerWriters returns where a call's command output should go, falling back
// to the Executor's own streams for whichever the listener declines to take.
func (e *Executor) listenerWriters(invocation Invocation) (io.Writer, io.Writer) {
	stdOut, stdErr := e.Stdout, e.Stderr
	if e.Listener == nil {
		return stdOut, stdErr
	}
	listenerOut, listenerErr := e.Listener.WriterFor(invocation)
	if listenerOut != nil {
		stdOut = listenerOut
	}
	if listenerErr != nil {
		stdErr = listenerErr
	}
	return stdOut, stdErr
}
