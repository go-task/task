package task

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"mvdan.cc/sh/v3/interp"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/slicesext"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/internal/summary"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

const (
	// MaximumTaskCall is the max number of times a task can be called.
	// This exists to prevent infinite loops on cyclic dependencies
	MaximumTaskCall = 1000
)

// MatchingTask represents a task that matches a given call. It includes the
// task itself and a list of wildcards that were matched.
type MatchingTask struct {
	Task      *ast.Task
	Wildcards []string
}

// Run runs Task
func (e *Executor) Run(ctx context.Context, calls ...*Call) error {
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

	g := &errgroup.Group{}
	if e.Failfast {
		g, ctx = errgroup.WithContext(ctx)
	}
	for _, c := range regularCalls {
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

func (e *Executor) splitRegularAndWatchCalls(calls ...*Call) (regularCalls []*Call, watchCalls []*Call, err error) {
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
	return regularCalls, watchCalls, err
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call *Call) error {
	t, err := e.FastCompiledTask(call)
	if err != nil {
		return err
	}
	if !shouldRunOnCurrentPlatform(t.Platforms) {
		e.Logger.VerboseOutf(logger.Yellow, `task: %q not for current platform - ignored\n`, call.Task)
		return nil
	}

	if err := e.areTaskRequiredVarsSet(t); err != nil {
		return err
	}

	t, err = e.CompiledTask(call)
	if err != nil {
		return err
	}

	if err := e.areTaskRequiredVarsAllowedValuesSet(t); err != nil {
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

	if err = e.startExecution(ctx, t, func(ctx context.Context) error {
		e.Logger.VerboseErrf(logger.Magenta, "task: %q started\n", call.Task)
		if err := e.runDeps(ctx, t); err != nil {
			return err
		}

		skipFingerprinting := e.ForceAll || (!call.Indirect && e.Force)
		if !skipFingerprinting {
			if err := ctx.Err(); err != nil {
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
				fingerprint.WithTempDir(e.TempDir.Fingerprint),
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

		for _, p := range t.Prompt {
			if p != "" && !e.Dry {
				if err := e.Logger.Prompt(logger.Yellow, p, "n", "y", "yes"); errors.Is(err, logger.ErrNoTerminal) {
					return &errors.TaskCancelledNoTerminalError{TaskName: call.Task}
				} else if errors.Is(err, logger.ErrPromptCancelled) {
					return &errors.TaskCancelledByUserError{TaskName: call.Task}
				} else if err != nil {
					return err
				}
			}
		}

		if err := e.mkdir(t); err != nil {
			e.Logger.Errf(logger.Red, "task: cannot make directory %q: %v\n", t.Dir, err)
		}

		var deferredExitCode uint8

		for i := range t.Cmds {
			if t.Cmds[i].Defer {
				defer e.runDeferred(t, call, i, t.Vars, &deferredExitCode)
				continue
			}

			if err := e.runCommand(ctx, t, call, i); err != nil {
				if err2 := e.statusOnError(t); err2 != nil {
					e.Logger.VerboseErrf(logger.Yellow, "task: error cleaning status on error: %v\n", err2)
				}

				var exitCode interp.ExitStatus
				if errors.As(err, &exitCode) {
					if t.IgnoreError {
						e.Logger.VerboseErrf(logger.Yellow, "task: task error ignored: %v\n", err)
						continue
					}
					deferredExitCode = uint8(exitCode)
				}

				return err
			}
		}
		e.Logger.VerboseErrf(logger.Magenta, "task: %q finished\n", call.Task)
		return nil
	}); err != nil {
		return &errors.TaskRunError{TaskName: t.Name(), Err: err}
	}

	return nil
}

func (e *Executor) mkdir(t *ast.Task) error {
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

func (e *Executor) runDeps(ctx context.Context, t *ast.Task) error {
	g := &errgroup.Group{}
	if e.Failfast || t.Failfast {
		g, ctx = errgroup.WithContext(ctx)
	}

	reacquire := e.releaseConcurrencyLimit()
	defer reacquire()

	for _, d := range t.Deps {
		g.Go(func() error {
			err := e.RunTask(ctx, &Call{Task: d.Task, Vars: d.Vars, Silent: d.Silent, Indirect: true})
			if err != nil {
				return err
			}
			return nil
		})
	}

	return g.Wait()
}

func (e *Executor) runDeferred(t *ast.Task, call *Call, i int, vars *ast.Vars, deferredExitCode *uint8) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := t.Cmds[i]
	cache := &templater.Cache{Vars: vars}
	extra := map[string]any{}

	if deferredExitCode != nil && *deferredExitCode > 0 {
		extra["EXIT_CODE"] = fmt.Sprintf("%d", *deferredExitCode)
	}

	cmd.Cmd = templater.ReplaceWithExtra(cmd.Cmd, cache, extra)
	cmd.Task = templater.ReplaceWithExtra(cmd.Task, cache, extra)
	cmd.Vars = templater.ReplaceVarsWithExtra(cmd.Vars, cache, extra)

	if err := e.runCommand(ctx, t, call, i); err != nil {
		e.Logger.VerboseErrf(logger.Yellow, "task: ignored error in deferred cmd: %s\n", err.Error())
	}
}

func (e *Executor) runCommand(ctx context.Context, t *ast.Task, call *Call, i int) error {
	cmd := t.Cmds[i]

	switch {
	case cmd.Task != "":
		reacquire := e.releaseConcurrencyLimit()
		defer reacquire()

		err := e.RunTask(ctx, &Call{Task: cmd.Task, Vars: cmd.Vars, Silent: cmd.Silent, Indirect: true})
		var exitCode interp.ExitStatus
		if errors.As(err, &exitCode) && cmd.IgnoreError {
			e.Logger.VerboseErrf(logger.Yellow, "task: [%s] task error ignored: %v\n", t.Name(), err)
			return nil
		}
		return err
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
		outputTemplater := &templater.Cache{Vars: vars}
		if err != nil {
			return fmt.Errorf("task: failed to get variables: %w", err)
		}
		stdOut, stdErr, closer := outputWrapper.WrapWriter(e.Stdout, e.Stderr, t.Prefix, outputTemplater)

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
		if closeErr := closer(err); closeErr != nil {
			e.Logger.Errf(logger.Red, "task: unable to close writer: %v\n", closeErr)
		}
		var exitCode interp.ExitStatus
		if errors.As(err, &exitCode) && cmd.IgnoreError {
			e.Logger.VerboseErrf(logger.Yellow, "task: [%s] command error ignored: %v\n", t.Name(), err)
			return nil
		}
		return err
	default:
		return nil
	}
}

func (e *Executor) startExecution(ctx context.Context, t *ast.Task, execute func(ctx context.Context) error) error {
	h, err := e.GetHash(t)
	if err != nil {
		return err
	}

	if h == "" || t.Watch {
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

// FindMatchingTasks returns a list of tasks that match the given call. A task
// matches a call if its name is equal to the call's task name or if it matches
// a wildcard pattern. The function returns a list of MatchingTask structs, each
// containing a task and a list of wildcards that were matched.
func (e *Executor) FindMatchingTasks(call *Call) []*MatchingTask {
	if call == nil {
		return nil
	}
	var matchingTasks []*MatchingTask
	// If there is a direct match, return it
	if task, ok := e.Taskfile.Tasks.Get(call.Task); ok {
		matchingTasks = append(matchingTasks, &MatchingTask{Task: task, Wildcards: nil})
		return matchingTasks
	}
	// Attempt a wildcard match
	for _, value := range e.Taskfile.Tasks.All(nil) {
		if match, wildcards := value.WildcardMatch(call.Task); match {
			matchingTasks = append(matchingTasks, &MatchingTask{
				Task:      value,
				Wildcards: wildcards,
			})
		}
	}
	return matchingTasks
}

// GetTask will return the task with the name matching the given call from the taskfile.
// If no task is found, it will search for tasks with a matching alias.
// If multiple tasks contain the same alias or no matches are found an error is returned.
func (e *Executor) GetTask(call *Call) (*ast.Task, error) {
	// Search for a matching task
	matchingTasks := e.FindMatchingTasks(call)
	if len(matchingTasks) > 0 {
		if call.Vars == nil {
			call.Vars = ast.NewVars()
		}
		call.Vars.Set("MATCH", ast.Var{Value: matchingTasks[0].Wildcards})
		return matchingTasks[0].Task, nil
	}

	// If didn't find one, search for a task with a matching alias
	var matchingTask *ast.Task
	var aliasedTasks []string
	for task := range e.Taskfile.Tasks.Values(nil) {
		if slices.Contains(task.Aliases, call.Task) {
			aliasedTasks = append(aliasedTasks, task.Task)
			matchingTask = task
		}
	}
	// If we found multiple tasks
	if len(aliasedTasks) > 1 {
		return nil, &errors.TaskNameConflictError{
			Call:      call.Task,
			TaskNames: aliasedTasks,
		}
	}
	// If we found no tasks
	if len(aliasedTasks) == 0 {
		didYouMean := ""
		if !e.DisableFuzzy {
			e.fuzzyModelOnce.Do(e.setupFuzzyModel)
			if e.fuzzyModel != nil {
				didYouMean = e.fuzzyModel.SpellCheck(call.Task)
			}
		}
		return nil, &errors.TaskNotFoundError{
			TaskName:   call.Task,
			DidYouMean: didYouMean,
		}
	}
	return matchingTask, nil
}

type FilterFunc func(task *ast.Task) bool

func (e *Executor) GetTaskList(filters ...FilterFunc) ([]*ast.Task, error) {
	tasks := make([]*ast.Task, 0, e.Taskfile.Tasks.Len())

	// Create an error group to wait for each task to be compiled
	var g errgroup.Group

	// Sort the tasks
	if e.TaskSorter == nil {
		e.TaskSorter = sort.AlphaNumericWithRootTasksFirst
	}

	// Filter tasks based on the given filter functions
	for task := range e.Taskfile.Tasks.Values(e.TaskSorter) {
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
		g.Go(func() error {
			compiledTask, err := e.CompiledTaskForTaskList(&Call{Task: tasks[i].Task})
			if err != nil {
				return err
			}
			tasks[i] = compiledTask
			return nil
		})
	}

	// Wait for all the go routines to finish
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// FilterOutNoDesc removes all tasks that do not contain a description.
func FilterOutNoDesc(task *ast.Task) bool {
	return task.Desc == ""
}

// FilterOutInternal removes all tasks that are marked as internal.
func FilterOutInternal(task *ast.Task) bool {
	return task.Internal
}

func shouldRunOnCurrentPlatform(platforms []*ast.Platform) bool {
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
