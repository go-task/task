package task

import (
	"io"
	"time"
)

// Invocation identifies one runtime call to a task. IDs are unique within an
// Executor, including repeated calls to the same task.
type Invocation struct {
	ID       uint64 // Unique call ID
	ParentID uint64 // Call that scheduled this one; zero for a requested root
	RootID   uint64 // Requested root this call descends from
	Task     string // Name as written in the Taskfile; the key GetTask takes
	Name     string // Display name: the task's label when it has one
}

// Result is how a call ended.
type Result uint8

const (
	// ResultSucceeded is the zero value, so a call that reported nothing needs
	// no further interpretation.
	ResultSucceeded Result = iota
	ResultFailed
	// ResultCanceled is a call interrupted before it could finish, by a failing
	// sibling under fail-fast or by the caller cancelling the context.
	ResultCanceled
	// ResultSkipped is a call Task chose not to run at all: the task is not for
	// the current platform, or its "if" condition was not met. Skipping is not a
	// failure, and Run returns no error for it.
	ResultSkipped
)

func (r Result) String() string {
	switch r {
	case ResultSucceeded:
		return "succeeded"
	case ResultFailed:
		return "failed"
	case ResultCanceled:
		return "canceled"
	case ResultSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// Started reports a call beginning execution. A call that is scheduled but
// never attempted, because an earlier root failed, is never started.
type Started struct {
	Invocation
	At time.Time
}

// Finished reports how a call ended. Every call that started is finished, as is
// every call that failed before it could start.
type Finished struct {
	Invocation
	Result Result
	// Err is the detail behind ResultFailed, and nil otherwise. For a task that
	// ran, it is an *errors.TaskRunError, whose TaskExitCode reports the exit
	// status of the command that failed.
	Err error
	At  time.Time
	// Duration is how long the call ran, and zero if it never started.
	Duration time.Duration
}

// Joined reports a call that waited on another call's execution rather than
// running its own, because the task is "run: once" or "run: when_changed". A
// joined call produces no output and takes its result from the owner.
type Joined struct {
	Invocation
	OwnerID uint64 // The call whose execution this one waited on
}

// Listener observes task execution. Assign one to Executor.Listener.
//
// Every field is optional: a zero Listener observes nothing and changes no
// behaviour. New fields may be added, so a client that sets only what it needs
// keeps working.
//
// Callbacks run on the goroutines executing the tasks, so implementations must
// be safe for concurrent use, and must return promptly: a callback that blocks
// holds up the task that reported it.
type Listener struct {
	// Scheduled reports a call Task intends to run. A call that is attempted is
	// always Finished, but a scheduled call may never be attempted, and then no
	// further event arrives for it.
	Scheduled func(Invocation)
	Started   func(Started)
	Finished  func(Finished)
	Joined    func(Joined)

	// OutputFor returns where a call's command output goes. A nil writer leaves
	// that stream on the Executor's own, which is what a listener that only
	// wants events should return.
	//
	// Output arrives as the command wrote it, escape sequences included, so a
	// client can render colour.
	OutputFor func(Invocation) (stdOut, stdErr io.Writer)

	// OwnsScreen says the client is drawing the display, so the Executor's own
	// Stdout, Stderr and Stdin are not usable. Task routes what it would have
	// printed through OutputFor, and refuses to run anything that needs the
	// terminal: interactive tasks, confirmation prompts and watch mode.
	OwnsScreen bool
}

func (e *Executor) notifyScheduled(invocation Invocation) {
	if e.Listener != nil && e.Listener.Scheduled != nil {
		e.Listener.Scheduled(invocation)
	}
}

func (e *Executor) notifyStarted(invocation Invocation, at time.Time) {
	if e.Listener != nil && e.Listener.Started != nil {
		e.Listener.Started(Started{Invocation: invocation, At: at})
	}
}

func (e *Executor) notifyFinished(finished Finished) {
	if e.Listener != nil && e.Listener.Finished != nil {
		e.Listener.Finished(finished)
	}
}

func (e *Executor) notifyJoined(invocation Invocation, ownerID uint64) {
	if e.Listener != nil && e.Listener.Joined != nil {
		e.Listener.Joined(Joined{Invocation: invocation, OwnerID: ownerID})
	}
}

// ownsScreen reports whether a client is drawing the display.
func (e *Executor) ownsScreen() bool {
	return e.Listener != nil && e.Listener.OwnsScreen
}

// listenerWriters returns where a call's command output should go, falling back
// to the Executor's own streams for whichever the listener declines to take.
func (e *Executor) listenerWriters(invocation Invocation) (io.Writer, io.Writer) {
	stdOut, stdErr := e.Stdout, e.Stderr
	if e.Listener == nil || e.Listener.OutputFor == nil {
		return stdOut, stdErr
	}
	listenerOut, listenerErr := e.Listener.OutputFor(invocation)
	if listenerOut != nil {
		stdOut = listenerOut
	}
	if listenerErr != nil {
		stdErr = listenerErr
	}
	return stdOut, stdErr
}
