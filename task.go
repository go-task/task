package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/go-task/task/v3/internal/compiler"
	compilerv2 "github.com/go-task/task/v3/internal/compiler/v2"
	compilerv3 "github.com/go-task/task/v3/internal/compiler/v3"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/summary"
	"github.com/go-task/task/v3/taskfile"
	"github.com/go-task/task/v3/taskfile/read"

	"golang.org/x/sync/errgroup"
)

const (
	// MaximumTaskCall is the max number of times a task can be called.
	// This exists to prevent infinite loops on cyclic dependencies
	MaximumTaskCall = 100
)

// Executor executes a Taskfile
type Executor struct {
	Taskfile *taskfile.Taskfile

	Dir         string
	Entrypoint  string
	Force       bool
	Watch       bool
	Verbose     bool
	Silent      bool
	Dry         bool
	Summary     bool
	Parallel    bool
	Color       bool
	Concurrency int

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Logger      *logger.Logger
	Compiler    compiler.Compiler
	Output      output.Output
	OutputStyle string

	taskvars *taskfile.Vars

	concurrencySemaphore chan struct{}
	taskCallCount        map[string]*int32
	mkdirMutexMap        map[string]*sync.Mutex
	executionHashes      map[string]context.Context
	executionHashesMutex sync.Mutex
}

// Run runs Task
func (e *Executor) Run(ctx context.Context, calls ...taskfile.Call) error {
	// check if given tasks exist
	for _, c := range calls {
		if _, ok := e.Taskfile.Tasks[c.Task]; !ok {
			// FIXME: move to the main package
			e.ListTasksWithDesc()
			return &taskNotFoundError{taskName: c.Task}
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

	if e.Watch {
		return e.watchTasks(calls...)
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, c := range calls {
		c := c
		if e.Parallel {
			g.Go(func() error { return e.RunTask(ctx, c) })
		} else {
			if err := e.RunTask(ctx, c); err != nil {
				return err
			}
		}
	}
	return g.Wait()
}

// Setup setups Executor's internal state
func (e *Executor) Setup() error {
	var err error
	e.Taskfile, err = read.Taskfile(e.Dir, e.Entrypoint)
	if err != nil {
		return err
	}

	v, err := e.Taskfile.ParsedVersion()
	if err != nil {
		return err
	}

	if v < 3.0 {
		e.taskvars, err = read.Taskvars(e.Dir)
		if err != nil {
			return err
		}
	}

	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}
	e.Logger = &logger.Logger{
		Stdout:  e.Stdout,
		Stderr:  e.Stderr,
		Verbose: e.Verbose,
		Color:   e.Color,
	}

	if v < 2 {
		return fmt.Errorf(`task: Taskfile versions prior to v2 are not supported anymore`)
	}

	// consider as equal to the greater version if round
	if v == 2.0 {
		v = 2.6
	}
	if v == 3.0 {
		v = 3.7
	}

	if v > 3.7 {
		return fmt.Errorf(`task: Taskfile versions greater than v3.7 not implemented in the version of Task`)
	}

	// Color available only on v3
	if v < 3 {
		e.Logger.Color = false
	}

	if v < 3 {
		e.Compiler = &compilerv2.CompilerV2{
			Dir:          e.Dir,
			Taskvars:     e.taskvars,
			TaskfileVars: e.Taskfile.Vars,
			Expansions:   e.Taskfile.Expansions,
			Logger:       e.Logger,
		}
	} else {
		e.Compiler = &compilerv3.CompilerV3{
			Dir:          e.Dir,
			TaskfileEnv:  e.Taskfile.Env,
			TaskfileVars: e.Taskfile.Vars,
			Logger:       e.Logger,
		}
	}

	if v >= 3.0 {
		env, err := read.Dotenv(e.Compiler, e.Taskfile, e.Dir)
		if err != nil {
			return err
		}

		err = env.Range(func(key string, value taskfile.Var) error {
			if _, ok := e.Taskfile.Env.Mapping[key]; !ok {
				e.Taskfile.Env.Set(key, value)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	if v < 2.1 && e.Taskfile.Output != "" {
		return fmt.Errorf(`task: Taskfile option "output" is only available starting on Taskfile version v2.1`)
	}
	if v < 2.2 && e.Taskfile.Includes.Len() > 0 {
		return fmt.Errorf(`task: Including Taskfiles is only available starting on Taskfile version v2.2`)
	}
	if v >= 3.0 && e.Taskfile.Expansions > 2 {
		return fmt.Errorf(`task: The "expansions" setting is not available anymore on v3.0`)
	}

	if e.OutputStyle != "" {
		e.Taskfile.Output = e.OutputStyle
	}
	switch e.Taskfile.Output {
	case "", "interleaved":
		e.Output = output.Interleaved{}
	case "group":
		e.Output = output.Group{}
	case "prefixed":
		e.Output = output.Prefixed{}
	default:
		return fmt.Errorf(`task: output option "%s" not recognized`, e.Taskfile.Output)
	}

	if e.Taskfile.Method == "" {
		if v >= 3 {
			e.Taskfile.Method = "checksum"
		} else {
			e.Taskfile.Method = "timestamp"
		}
	}

	if v <= 2.1 {
		err := errors.New(`task: Taskfile option "ignore_error" is only available starting on Taskfile version v2.1`)

		for _, task := range e.Taskfile.Tasks {
			if task.IgnoreError {
				return err
			}
			for _, cmd := range task.Cmds {
				if cmd.IgnoreError {
					return err
				}
			}
		}
	}

	if v < 2.6 {
		for _, task := range e.Taskfile.Tasks {
			if len(task.Preconditions) > 0 {
				return errors.New(`task: Task option "preconditions" is only available starting on Taskfile version v2.6`)
			}
		}
	}

	if v < 3 {
		err := e.Taskfile.Includes.Range(func(_ string, taskfile taskfile.IncludedTaskfile) error {
			if taskfile.AdvancedImport {
				return errors.New(`task: Import with additional parameters is only available starting on Taskfile version v3`)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	if v < 3.7 {
		if e.Taskfile.Run != "" {
			return errors.New(`task: Setting the "run" type is only available starting on Taskfile version v3.7`)
		}

		for _, task := range e.Taskfile.Tasks {
			if task.Run != "" {
				return errors.New(`task: Setting the "run" type is only available starting on Taskfile version v3.7`)
			}
		}
	}

	if e.Taskfile.Run == "" {
		e.Taskfile.Run = "always"
	}

	e.executionHashes = make(map[string]context.Context)

	e.taskCallCount = make(map[string]*int32, len(e.Taskfile.Tasks))
	e.mkdirMutexMap = make(map[string]*sync.Mutex, len(e.Taskfile.Tasks))
	for k := range e.Taskfile.Tasks {
		e.taskCallCount[k] = new(int32)
		e.mkdirMutexMap[k] = &sync.Mutex{}
	}

	if e.Concurrency > 0 {
		e.concurrencySemaphore = make(chan struct{}, e.Concurrency)
	}
	return nil
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call taskfile.Call) error {
	t, err := e.CompiledTask(call)
	if err != nil {
		return err
	}
	if !e.Watch && atomic.AddInt32(e.taskCallCount[call.Task], 1) >= MaximumTaskCall {
		return &MaximumTaskCallExceededError{task: call.Task}
	}

	release := e.acquireConcurrencyLimit()
	defer release()

	return e.startExecution(ctx, t, func(ctx context.Context) error {
		e.Logger.VerboseErrf(logger.Magenta, `task: "%s" started`, call.Task)
		if err := e.runDeps(ctx, t); err != nil {
			return err
		}

		if !e.Force {
			if err := ctx.Err(); err != nil {
				return err
			}

			preCondMet, err := e.areTaskPreconditionsMet(ctx, t)
			if err != nil {
				return err
			}

			upToDate, err := e.isTaskUpToDate(ctx, t)
			if err != nil {
				return err
			}

			if upToDate && preCondMet {
				if !e.Silent {
					e.Logger.Errf(logger.Magenta, `task: Task "%s" is up to date`, t.Name())
				}
				return nil
			}
		}

		if err := e.mkdir(t); err != nil {
			e.Logger.Errf(logger.Red, "task: cannot make directory %q: %v", t.Dir, err)
		}

		for i := range t.Cmds {
			if t.Cmds[i].Defer {
				defer e.runDeferred(t, call, i)
				continue
			}

			if err := e.runCommand(ctx, t, call, i); err != nil {
				if err2 := e.statusOnError(t); err2 != nil {
					e.Logger.VerboseErrf(logger.Yellow, "task: error cleaning status on error: %v", err2)
				}

				if execext.IsExitError(err) && t.IgnoreError {
					e.Logger.VerboseErrf(logger.Yellow, "task: task error ignored: %v", err)
					continue
				}

				return &taskRunError{t.Task, err}
			}
		}
		e.Logger.VerboseErrf(logger.Magenta, `task: "%s" finished`, call.Task)
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
		if err := os.MkdirAll(t.Dir, 0755); err != nil {
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
			err := e.RunTask(ctx, taskfile.Call{Task: d.Task, Vars: d.Vars})
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
		e.Logger.VerboseErrf(logger.Yellow, `task: ignored error in deferred cmd: %s`, err.Error())
	}
}

func (e *Executor) runCommand(ctx context.Context, t *taskfile.Task, call taskfile.Call, i int) error {
	cmd := t.Cmds[i]

	switch {
	case cmd.Task != "":
		reacquire := e.releaseConcurrencyLimit()
		defer reacquire()

		err := e.RunTask(ctx, taskfile.Call{Task: cmd.Task, Vars: cmd.Vars})
		if err != nil {
			return err
		}
		return nil
	case cmd.Cmd != "":
		if e.Verbose || (!cmd.Silent && !t.Silent && !e.Taskfile.Silent && !e.Silent) {
			e.Logger.Errf(logger.Green, "task: [%s] %s", t.Name(), cmd.Cmd)
		}

		if e.Dry {
			return nil
		}

		outputWrapper := e.Output
		if t.Interactive {
			outputWrapper = output.Interleaved{}
		}
		stdOut := outputWrapper.WrapWriter(e.Stdout, t.Prefix)
		stdErr := outputWrapper.WrapWriter(e.Stderr, t.Prefix)

		defer func() {
			if _, ok := stdOut.(*os.File); !ok {
				if closer, ok := stdOut.(io.Closer); ok {
					closer.Close()
				}
			}
			if _, ok := stdErr.(*os.File); !ok {
				if closer, ok := stdErr.(io.Closer); ok {
					closer.Close()
				}
			}
		}()

		err := execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command: cmd.Cmd,
			Dir:     t.Dir,
			Env:     getEnviron(t),
			Stdin:   e.Stdin,
			Stdout:  stdOut,
			Stderr:  stdErr,
		})
		if execext.IsExitError(err) && cmd.IgnoreError {
			e.Logger.VerboseErrf(logger.Yellow, "task: [%s] command error ignored: %v", t.Name(), err)
			return nil
		}
		return err
	default:
		return nil
	}
}

func getEnviron(t *taskfile.Task) []string {
	if t.Env == nil {
		return nil
	}

	environ := os.Environ()

	for k, v := range t.Env.ToCacheMap() {
		str, isString := v.(string)
		if !isString {
			continue
		}

		if _, alreadySet := os.LookupEnv(k); alreadySet {
			continue
		}

		environ = append(environ, fmt.Sprintf("%s=%s", k, str))
	}

	return environ
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
	otherExecutionCtx, ok := e.executionHashes[h]

	if ok {
		e.executionHashesMutex.Unlock()
		e.Logger.VerboseErrf(logger.Magenta, "task: skipping execution of task: %s", h)
		<-otherExecutionCtx.Done()
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	e.executionHashes[h] = ctx
	e.executionHashesMutex.Unlock()

	return execute(ctx)
}
