package output

import (
	"context"
	"fmt"
	"io"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/internal/term"
	"github.com/go-task/task/v3/taskfile/ast"
)

const (
	systemTaskName   = "Task messages"
	maxTaskOutputLen = 10 << 20
)

type TUI struct {
	logger       *logger.Logger
	input        io.Reader
	output       io.Writer
	hideInternal bool
	statusLabels bool

	mutex   sync.RWMutex
	program *tea.Program

	outputMutex  sync.Mutex
	pending      map[uint64]pendingOutput
	outputQueued bool
}

type pendingOutput struct {
	name string
	data string
}

func NewTUI(log *logger.Logger, options ast.OutputTUI) (*TUI, error) {
	if !log.AssumeTerm && !term.IsTerminal() {
		return nil, fmt.Errorf(`task: output style "tui" requires an interactive terminal`)
	}
	statusLabels := false
	switch options.Status {
	case "", "icons":
	case "labels":
		statusLabels = true
	default:
		return nil, fmt.Errorf(`task: invalid TUI status style %q: expected "icons" or "labels"`, options.Status)
	}
	return &TUI{
		logger:       log,
		input:        log.Stdin,
		output:       log.Stdout,
		hideInternal: options.HideInternal,
		statusLabels: statusLabels,
		pending:      make(map[uint64]pendingOutput),
	}, nil
}

// WrapWriter satisfies Output. Executor calls WrapWriterForTask so output can
// be associated with a specific invocation; other callers use the system log.
func (t *TUI) WrapWriter(_ io.Writer, _ io.Writer, _ string, _ *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	w := &tuiWriter{tui: t, name: systemTaskName}
	return w, w, func(error) error { return nil }
}

func (t *TUI) WrapWriterForTask(_ io.Writer, _ io.Writer, task TaskInvocation, _ *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	w := &tuiWriter{tui: t, id: task.ID, name: task.Name}
	return w, w, func(error) error { return nil }
}

func (t *TUI) TaskScheduled(task TaskInvocation) {
	t.send(taskScheduledMsg{task: task})
}

func (t *TUI) TaskStarted(task TaskInvocation) {
	t.send(taskStartedMsg{task: task})
}

func (t *TUI) TaskFinished(id uint64, err error) {
	t.send(taskFinishedMsg{id: id, err: err})
}

func (t *TUI) TaskJoined(id, ownerID uint64) {
	t.send(taskJoinedMsg{id: id, ownerID: ownerID})
}

func (t *TUI) Run(ctx context.Context, run func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := newTUIModel(cancel, t.hideInternal)
	model.statusLabels = t.statusLabels
	program := tea.NewProgram(
		model,
		tea.WithInput(t.input),
		tea.WithOutput(t.output),
	)

	t.mutex.Lock()
	t.program = program
	t.mutex.Unlock()

	oldStdout, oldStderr := t.logger.Stdout, t.logger.Stderr
	systemWriter := &tuiWriter{tui: t, name: systemTaskName}
	t.logger.Stdout, t.logger.Stderr = systemWriter, systemWriter
	defer func() {
		t.logger.Stdout, t.logger.Stderr = oldStdout, oldStderr
		t.mutex.Lock()
		t.program = nil
		t.mutex.Unlock()
	}()

	done := make(chan error, 1)
	go func() {
		<-ctx.Done()
		program.Send(tea.Interrupt())
	}()
	go func() {
		err := run(ctx)
		done <- err
		t.send(executionDoneMsg{err: err})
	}()

	_, uiErr := program.Run()
	cancel()
	runErr := <-done
	if uiErr != nil {
		return fmt.Errorf("task: TUI failed: %w", uiErr)
	}
	return runErr
}

func (t *TUI) send(msg tea.Msg) {
	t.mutex.RLock()
	program := t.program
	t.mutex.RUnlock()
	if program != nil {
		program.Send(msg)
	}
}

func (t *TUI) enqueueOutput(id uint64, name, data string) {
	t.outputMutex.Lock()
	pending := t.pending[id]
	pending.name = name
	pending.data += data
	t.pending[id] = pending
	if t.outputQueued {
		t.outputMutex.Unlock()
		return
	}
	t.outputQueued = true
	t.outputMutex.Unlock()

	// Sending asynchronously lets bursts of command output collapse into one
	// model update instead of rebuilding the viewport for every pipe write.
	go t.send(outputReadyMsg{tui: t})
}

func (t *TUI) drainOutput() map[uint64]pendingOutput {
	t.outputMutex.Lock()
	defer t.outputMutex.Unlock()
	output := t.pending
	t.pending = make(map[uint64]pendingOutput)
	t.outputQueued = false
	return output
}

type tuiWriter struct {
	tui  *TUI
	id   uint64
	name string
}

func (w *tuiWriter) Write(p []byte) (int, error) {
	data := string(append([]byte(nil), p...))
	w.tui.enqueueOutput(w.id, w.name, data)
	return len(p), nil
}
