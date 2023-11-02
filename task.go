package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/compiler"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/slicesext"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/internal/summary"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"

	"github.com/sajari/fuzzy"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

const (
	// MaximumTaskCall is the max number of times a task can be called.
	// This exists to prevent infinite loops on cyclic dependencies
	MaximumTaskCall = 1000
)

// Executor executes a Taskfile
type Executor struct {
	Taskfile *taskfile.Taskfile

	Dir         string
	TempDir     string
	Entrypoint  string
	Force       bool
	ForceAll    bool
	Insecure    bool
	Download    bool
	Offline     bool
	Timeout     time.Duration
	Watch       bool
	Verbose     bool
	Silent      bool
	AssumeYes   bool
	AssumeTerm  bool // Used for testing
	Dry         bool
	Summary     bool
	Parallel    bool
	Color       bool
	Concurrency int
	Interval    time.Duration

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Logger         *logger.Logger
	Compiler       compiler.Compiler
	Output         output.Output
	OutputStyle    taskfile.Output
	TaskSorter     sort.TaskSorter
	UserWorkingDir string

	taskvars   *taskfile.Vars
	fuzzyModel *fuzzy.Model

	concurrencySemaphore chan struct{}
	taskCallCount        map[string]*int32
	mkdirMutexMap        map[string]*sync.Mutex
	executionHashes      map[string]context.Context
	executionHashesMutex sync.Mutex
}

// Run runs Task
func (e *Executor) Run(ctx context.Context, calls ...taskfile.Call) error {
	// check if given tasks exist
	for _, call := range calls {
		task, err := e.GetTask(call)
		if err != nil {
			if _, ok := err.(*errors.TaskNotFoundError); ok {
				if _, err := e.ListTasks(ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
					return err
				}
			}
			return err
		}

		if task.Internal {
			if _, ok := err.(*errors.TaskNotFoundError); ok {
				if _, err := e.ListTasks(ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
					return err
				}
			}
			return &errors.TaskInternalError{TaskName: call.Task}
		}
	}

	if e.Summary {
		for i, c := range calls {
			compiledTask, err := e.FastCompiledTask(c)
			if err != nil {
				return nil
			}
			summary.PrintSpaceBetweenSummaries(e.Logger, i)
			summary.PrintTask(e.Logger, compiledTask)
		}
		return nil
	}

	regularCalls, watchCalls, err := e.splitRegularAndWatchCalls(calls...)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, c := range regularCalls {
		c := c
		if e.Parallel {
			g.Go(func() error { return e.RunTask(ctx, c) })
		} else {
			if err := e.RunTask(ctx, c); err != nil {
				return err
			}
		}
	}
	if err := g.Wait(); err != nil {
		return err
	}

	if len(watchCalls) > 0 {
		return e.watchTasks(watchCalls...)
	}

	return nil
}

func (e *Executor) splitRegularAndWatchCalls(calls ...taskfile.Call) (regularCalls []taskfile.Call, watchCalls []taskfile.Call, err error) {
	for _, c := range calls {
		t, err := e.GetTask(c)
		if err != nil {
			return nil, nil, err
		}

		if e.Watch || t.Watch {
			watchCalls = append(watchCalls, c)
		} else {
			regularCalls = append(regularCalls, c)
		}
	}
	return
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call taskfile.Call) error {
	t, err := e.FastCompiledTask(call)
	if err != nil {
		return err
	}
	if !shouldRunOnCurrentPlatform(t.Platforms) {
		e.Logger.VerboseOutf(logger.Yellow, `task: %q not for current platform - ignored\n`, call.Task)
		return nil
	}

	t, err = e.CompiledTask(call)
	if err != nil {
		return err
	}
	if !e.Watch && atomic.AddInt32(e.taskCallCount[t.Task], 1) >= MaximumTaskCall {
		return &errors.TaskCalledTooManyTimesError{
			TaskName:        t.Task,
			MaximumTaskCall: MaximumTaskCall,
		}
	}

	release := e.acquireConcurrencyLimit()
	defer release()

	if t.Prompt != "" {
		if err := e.Logger.Prompt(logger.Yellow, t.Prompt, "n", "y", "yes"); errors.Is(err, logger.ErrNoTerminal) {
			return &errors.TaskCancelledNoTerminalError{TaskName: call.Task}
		} else if errors.Is(err, logger.ErrPromptCancelled) {
			return &errors.TaskCancelledByUserError{TaskName: call.Task}
		} else if err != nil {
			return err
		}
	}

	return e.startExecution(ctx, t, func(ctx context.Context) error {
		e.Logger.VerboseErrf(logger.Magenta, "task: %q started\n", call.Task)
		if err := e.runDeps(ctx, t); err != nil {
			return err
		}

		skipFingerprinting := e.ForceAll || (call.Direct && e.Force)
		if !skipFingerprinting {
			if err := ctx.Err(); err != nil {
				return err
			}

			if err := e.areTaskRequiredVarsSet(ctx, t, call); err != nil {
				return err
			}

			preCondMet, err := e.areTaskPreconditionsMet(ctx, t)
			if err != nil {
				return err
			}

			// Get the fingerprinting method to use
			method := e.Taskfile.Method
			if t.Method != "" {
				method = t.Method
			}

			upToDate, err := fingerprint.IsTaskUpToDate(ctx, t,
				fingerprint.WithMethod(method),
				fingerprint.WithTempDir(e.TempDir),
				fingerprint.WithDry(e.Dry),
				fingerprint.WithLogger(e.Logger),
			)
			if err != nil {
				return err
			}

			if upToDate && preCondMet {
				if e.Verbose || (!call.Silent && !t.Silent && !e.Taskfile.Silent && !e.Silent) {
					e.Logger.Errf(logger.Magenta, "task: Task %q is up to date\n", t.Name())
				}
				return nil
			}
		}

		if err := e.mkdir(t); err != nil {
			e.Logger.Errf(logger.Red, "task: cannot make directory %q: %v\n", t.Dir, err)
		}

		for i := range t.Cmds {
			if t.Cmds[i].Defer {
				defer e.runDeferred(t, call, i)
				continue
			}

			if err := e.runCommand(ctx, t, call, i); err != nil {
				if err2 := e.statusOnError(t); err2 != nil {
					e.Logger.VerboseErrf(logger.Yellow, "task: error cleaning status on error: %v\n", err2)
				}

				if execext.IsExitError(err) && t.IgnoreError {
					e.Logger.VerboseErrf(logger.Yellow, "task: task error ignored: %v\n", err)
					continue
				}

				if !call.Direct {
					return err
				}

				return &errors.TaskRunError{TaskName: t.Task, Err: err}
			}
		}
		e.Logger.VerboseErrf(logger.Magenta, "task: %q finished\n", call.Task)
		return nil
	})
}

func (e *Executor) mkdir(t *taskfile.Task) error {
	if t.Dir == "" {
		return nil
	}

	mutex := e.mkdirMutexMap[t.Task]
	mutex.Lock()
	defer mutex.Unlock()

	if _, err := os.Stat(t.Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(t.Dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) runDeps(ctx context.Context, t *taskfile.Task) error {
	g, ctx := errgroup.WithContext(ctx)

	reacquire := e.releaseConcurrencyLimit()
	defer reacquire()

	for _, d := range t.Deps {
		d := d
		g.Go(func() error {
			err := e.RunTask(ctx, taskfile.Call{Task: d.Task, Vars: d.Vars, Silent: d.Silent})
			if err != nil {
				return err
			}
			return nil
		})
	}

	return g.Wait()
}

func (e *Executor) runDeferred(t *taskfile.Task, call taskfile.Call, i int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := e.runCommand(ctx, t, call, i); err != nil {
		e.Logger.VerboseErrf(logger.Yellow, "task: ignored error in deferred cmd: %s\n", err.Error())
	}
}

func (e *Executor) runCommand(ctx context.Context, t *taskfile.Task, call taskfile.Call, i int) error {
	cmd := t.Cmds[i]

	switch {
	case cmd.Task != "":
		reacquire := e.releaseConcurrencyLimit()
		defer reacquire()

		err := e.RunTask(ctx, taskfile.Call{Task: cmd.Task, Vars: cmd.Vars, Silent: cmd.Silent})
		if err != nil {
			return err
		}
		return nil
	case cmd.Cmd != "":
		if !shouldRunOnCurrentPlatform(cmd.Platforms) {
			e.Logger.VerboseOutf(logger.Yellow, "task: [%s] %s not for current platform - ignored\n", t.Name(), cmd.Cmd)
			return nil
		}

		if e.Verbose || (!call.Silent && !cmd.Silent && !t.Silent && !e.Taskfile.Silent && !e.Silent) {
			e.Logger.Errf(logger.Green, "task: [%s] %s\n", t.Name(), cmd.Cmd)
		}

		if e.Dry {
			return nil
		}

		outputWrapper := e.Output
		if t.Interactive {
			outputWrapper = output.Interleaved{}
		}
		vars, err := e.Compiler.FastGetVariables(t, call)
		outputTemplater := &templater.Templater{Vars: vars, RemoveNoValue: true}
		if err != nil {
			return fmt.Errorf("task: failed to get variables: %w", err)
		}
		stdOut, stdErr, close := outputWrapper.WrapWriter(e.Stdout, e.Stderr, t.Prefix, outputTemplater)

		err = execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command:   cmd.Cmd,
			Dir:       t.Dir,
			Env:       env.Get(t),
			PosixOpts: slicesext.UniqueJoin(e.Taskfile.Set, t.Set, cmd.Set),
			BashOpts:  slicesext.UniqueJoin(e.Taskfile.Shopt, t.Shopt, cmd.Shopt),
			Stdin:     e.Stdin,
			Stdout:    stdOut,
			Stderr:    stdErr,
		})
		if closeErr := close(err); closeErr != nil {
			e.Logger.Errf(logger.Red, "task: unable to close writer: %v\n", closeErr)
		}
		if execext.IsExitError(err) && cmd.IgnoreError {
			e.Logger.VerboseErrf(logger.Yellow, "task: [%s] command error ignored: %v\n", t.Name(), err)
			return nil
		}
		return err
	default:
		return nil
	}
}

func (e *Executor) startExecution(ctx context.Context, t *taskfile.Task, execute func(ctx context.Context) error) error {
	h, err := e.GetHash(t)
	if err != nil {
		return err
	}

	if h == "" {
		return execute(ctx)
	}

	e.executionHashesMutex.Lock()

	if otherExecutionCtx, ok := e.executionHashes[h]; ok {
		e.executionHashesMutex.Unlock()
		e.Logger.VerboseErrf(logger.Magenta, "task: skipping execution of task: %s\n", h)

		// Release our execution slot to avoid blocking other tasks while we wait
		reacquire := e.releaseConcurrencyLimit()
		defer reacquire()

		<-otherExecutionCtx.Done()
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	e.executionHashes[h] = ctx
	e.executionHashesMutex.Unlock()

	return execute(ctx)
}

// GetTask will return the task with the name matching the given call from the taskfile.
// If no task is found, it will search for tasks with a matching alias.
// If multiple tasks contain the same alias or no matches are found an error is returned.
func (e *Executor) GetTask(call taskfile.Call) (*taskfile.Task, error) {
	// Search for a matching task
	matchingTask := e.Taskfile.Tasks.Get(call.Task)
	if matchingTask != nil {
		return matchingTask, nil
	}

	// If didn't find one, search for a task with a matching alias
	var aliasedTasks []string
	for _, task := range e.Taskfile.Tasks.Values() {
		if slices.Contains(task.Aliases, call.Task) {
			aliasedTasks = append(aliasedTasks, task.Task)
			matchingTask = task
		}
	}
	// If we found multiple tasks
	if len(aliasedTasks) > 1 {
		return nil, &errors.TaskNameConflictError{
			AliasName: call.Task,
			TaskNames: aliasedTasks,
		}
	}
	// If we found no tasks
	if len(aliasedTasks) == 0 {
		didYouMean := ""
		if e.fuzzyModel != nil {
			didYouMean = e.fuzzyModel.SpellCheck(call.Task)
		}
		return nil, &errors.TaskNotFoundError{
			TaskName:   call.Task,
			DidYouMean: didYouMean,
		}
	}

	return matchingTask, nil
}

type FilterFunc func(task *taskfile.Task) bool

func (e *Executor) GetTaskList(filters ...FilterFunc) ([]*taskfile.Task, error) {
	tasks := make([]*taskfile.Task, 0, e.Taskfile.Tasks.Len())

	// Create an error group to wait for each task to be compiled
	var g errgroup.Group

	// Filter tasks based on the given filter functions
	for _, task := range e.Taskfile.Tasks.Values() {
		var shouldFilter bool
		for _, filter := range filters {
			if filter(task) {
				shouldFilter = true
			}
		}
		if !shouldFilter {
			tasks = append(tasks, task)
		}
	}

	// Compile the list of tasks
	for i := range tasks {
		idx := i
		task := tasks[idx]
		g.Go(func() error {
			compiledTask, err := e.FastCompiledTask(taskfile.Call{Task: task.Task})
			if err == nil {
				task = compiledTask
			}
			tasks[idx] = compiledTask
			return nil
		})
	}

	// Wait for all the go routines to finish
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Sort the tasks
	if e.TaskSorter == nil {
		e.TaskSorter = &sort.AlphaNumericWithRootTasksFirst{}
	}
	e.TaskSorter.Sort(tasks)

	return tasks, nil
}

// FilterOutNoDesc removes all tasks that do not contain a description.
func FilterOutNoDesc(task *taskfile.Task) bool {
	return task.Desc == ""
}

// FilterOutInternal removes all tasks that are marked as internal.
func FilterOutInternal(task *taskfile.Task) bool {
	return task.Internal
}

func shouldRunOnCurrentPlatform(platforms []*taskfile.Platform) bool {
	if len(platforms) == 0 {
		return true
	}
	for _, p := range platforms {
		if (p.OS == "" || p.OS == runtime.GOOS) && (p.Arch == "" || p.Arch == runtime.GOARCH) {
			return true
		}
	}
	return false
}
