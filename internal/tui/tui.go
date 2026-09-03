package tui

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/term"
)

const (
	systemTaskName   = "Task messages"
	maxTaskOutputLen = 10 << 20
)

// taskInvocation aliases the executor type so the rest of this package can keep
// using "task" as a local variable name without shadowing the package.
type taskInvocation = task.Invocation

// UI captures task lifecycle and output events for the interactive interface.
type UI struct {
	logger        *logger.Logger
	input         io.Reader
	output        io.Writer
	statusLabels  bool
	taskNavigator tuiTaskNavigator

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

// Options configures the execution dashboard.
type Options struct {
	Status        string
	TaskNavigator string
}

// New creates a terminal interface using the logger's input and output streams.
func New(log *logger.Logger, options Options) (*UI, error) {
	if !log.AssumeTerm && !term.IsTerminal() {
		return nil, fmt.Errorf("task: --tui requires an interactive terminal")
	}
	statusLabels := false
	switch options.Status {
	case "", "icons":
	case "labels":
		statusLabels = true
	default:
		return nil, fmt.Errorf(`task: invalid TUI status style %q: expected "icons" or "labels"`, options.Status)
	}
	taskNavigator := taskNavigatorTree
	switch options.TaskNavigator {
	case "", "tree":
	case "list":
		taskNavigator = taskNavigatorList
	default:
		return nil, fmt.Errorf(`task: invalid TUI task navigator %q: expected "list" or "tree"`, options.TaskNavigator)
	}
	return &UI{
		logger:        log,
		input:         log.Stdin,
		output:        log.Stdout,
		statusLabels:  statusLabels,
		taskNavigator: taskNavigator,
		pending:       make(map[uint64]pendingOutput),
	}, nil
}

// WriterFor routes a task's command output into that task's own pane.
func (t *UI) WriterFor(invocation task.Invocation) (io.Writer, io.Writer) {
	w := &tuiWriter{ui: t, id: invocation.ID, name: invocation.Name}
	return w, w
}

func (t *UI) TaskScheduled(invocation task.Invocation) {
	t.send(taskScheduledMsg{task: invocation})
}

func (t *UI) TaskStarted(invocation task.Invocation) {
	t.send(taskStartedMsg{task: invocation})
}

func (t *UI) TaskFinished(id uint64, err error) {
	t.send(taskFinishedMsg{id: id, err: err})
}

func (t *UI) TaskJoined(id, ownerID uint64) {
	t.send(taskJoinedMsg{id: id, ownerID: ownerID})
}

func (*UI) OwnsTerminal() bool { return true }

// Run opens the launcher when calls is empty, or starts the calls immediately.
func (t *UI) Run(ctx context.Context, executor *task.Executor, calls []*task.Call) error {
	sessionCtx, cancelSession := context.WithCancel(ctx)
	defer cancelSession()

	loadLauncher := func() (launcherModel, error) {
		tasks, err := executor.GetTaskList(task.FilterOutInternal)
		if err != nil {
			return launcherModel{}, err
		}
		return newLauncherModel(tasks), nil
	}
	var launcher launcherModel
	if len(calls) == 0 {
		var err error
		launcher, err = loadLauncher()
		if err != nil {
			return err
		}
	} else {
		// Resolve the requested tasks before the alt screen opens, so an unknown
		// task name reports itself on the terminal instead of inside a dashboard
		// the user then has to quit.
		for _, call := range calls {
			if _, err := executor.GetTask(call); err != nil {
				return err
			}
		}
	}

	execution := newTUIModel(func() {})
	execution.statusLabels = t.statusLabels
	execution.taskNavigator = t.taskNavigator
	execution.canReturnToLauncher = true

	var runs sync.WaitGroup
	var started atomic.Bool
	var resultMutex sync.Mutex
	var lastRunErr error
	programReady := make(chan struct{})
	start := func(selectedCalls []*task.Call) context.CancelFunc {
		runCtx, cancelRun := context.WithCancel(sessionCtx)
		started.Store(true)
		// Each launcher selection is an independent run. Without this, the
		// second run of a "run: once" task joins the first run's finished
		// execution and returns its result without executing anything.
		executor.ResetRunState()
		runs.Go(func() {
			<-programReady
			err := executor.Run(runCtx, selectedCalls...)
			resultMutex.Lock()
			lastRunErr = err
			resultMutex.Unlock()
			t.send(executionDoneMsg{ui: t, err: err})
		})
		return cancelRun
	}

	var normalTask string
	model := newAppModel(
		launcher,
		execution,
		len(calls) == 0,
		loadLauncher,
		func(names []string) context.CancelFunc {
			selectedCalls := make([]*task.Call, len(names))
			for i, name := range names {
				selectedCalls[i] = &task.Call{Task: name}
			}
			return start(selectedCalls)
		},
		func(name string) { normalTask = name },
	)
	if len(calls) > 0 {
		model.execution.cancel = start(calls)
	}
	program := tea.NewProgram(
		model,
		tea.WithInput(t.input),
		tea.WithOutput(t.output),
		tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if _, ok := msg.(tea.InterruptMsg); ok {
				return interruptRequestedMsg{}
			}
			return msg
		}),
	)

	t.mutex.Lock()
	t.program = program
	t.mutex.Unlock()

	executor.Listener = t

	oldStdout, oldStderr := t.logger.Stdout, t.logger.Stderr
	systemWriter := &tuiWriter{ui: t, name: systemTaskName}
	t.logger.Stdout, t.logger.Stderr = systemWriter, systemWriter
	var restoreOnce sync.Once
	restore := func() {
		restoreOnce.Do(func() {
			executor.Listener = nil
			t.logger.Stdout, t.logger.Stderr = oldStdout, oldStderr
			t.mutex.Lock()
			t.program = nil
			t.mutex.Unlock()
		})
	}
	defer restore()
	close(programReady)

	go func() {
		<-sessionCtx.Done()
		program.Send(interruptRequestedMsg{})
	}()
	finalModel, uiErr := program.Run()
	cancelSession()
	runs.Wait()
	if uiErr != nil {
		return fmt.Errorf("task: TUI failed: %w", uiErr)
	}
	finalApp := finalModel.(appModel)
	if finalApp.err != nil {
		return finalApp.err
	}
	if normalTask != "" {
		restore()
		return executor.Run(ctx, &task.Call{Task: normalTask})
	}
	if !started.Load() || finalApp.page == launcherPage {
		return nil
	}
	resultMutex.Lock()
	defer resultMutex.Unlock()
	return lastRunErr
}

func (t *UI) send(msg tea.Msg) {
	t.mutex.RLock()
	program := t.program
	t.mutex.RUnlock()
	if program != nil {
		program.Send(msg)
	}
}

func (t *UI) enqueueOutput(id uint64, name, data string) {
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
	go t.send(outputReadyMsg{ui: t})
}

func (t *UI) drainOutput() map[uint64]pendingOutput {
	t.outputMutex.Lock()
	defer t.outputMutex.Unlock()
	output := t.pending
	t.pending = make(map[uint64]pendingOutput)
	t.outputQueued = false
	return output
}

type tuiWriter struct {
	ui   *UI
	id   uint64
	name string
}

func (w *tuiWriter) Write(p []byte) (int, error) {
	data := string(append([]byte(nil), p...))
	w.ui.enqueueOutput(w.id, w.name, data)
	return len(p), nil
}
