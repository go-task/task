package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/go-task/task/internal/compiler"
	compilerv1 "github.com/go-task/task/internal/compiler/v1"
	compilerv2 "github.com/go-task/task/internal/compiler/v2"
	"github.com/go-task/task/internal/execext"
	"github.com/go-task/task/internal/logger"
	"github.com/go-task/task/internal/output"
	"github.com/go-task/task/internal/taskfile"
	"github.com/go-task/task/internal/taskfile/read"
	"github.com/go-task/task/internal/taskfile/version"

	"github.com/Masterminds/semver"
	"golang.org/x/sync/errgroup"
	"mvdan.cc/sh/interp"
)

const (
	// MaximumTaskCall is the max number of times a task can be called.
	// This exists to prevent infinite loops on cyclic dependencies
	MaximumTaskCall = 100
)

// Executor executes a Taskfile
type Executor struct {
	Taskfile *taskfile.Taskfile
	Dir      string
	Force    bool
	Watch    bool
	Verbose  bool
	Silent   bool
	Dry      bool

	Context context.Context

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Logger   *logger.Logger
	Compiler compiler.Compiler
	Output   output.Output

	taskvars taskfile.Vars

	taskCallCount map[string]*int32
}

// Run runs Task
func (e *Executor) Run(calls ...taskfile.Call) error {
	// check if given tasks exist
	for _, c := range calls {
		if _, ok := e.Taskfile.Tasks[c.Task]; !ok {
			// FIXME: move to the main package
			e.PrintTasksHelp()
			return &taskNotFoundError{taskName: c.Task}
		}
	}

	if e.Watch {
		return e.watchTasks(calls...)
	}

	for _, c := range calls {
		if err := e.RunTask(e.Context, c); err != nil {
			return err
		}
	}
	return nil
}

// Setup setups Executor's internal state
func (e *Executor) Setup() error {
	var err error
	e.Taskfile, err = read.Taskfile(e.Dir)
	if err != nil {
		return err
	}
	e.taskvars, err = read.Taskvars(e.Dir)
	if err != nil {
		return err
	}

	v, err := semver.NewConstraint(e.Taskfile.Version)
	if err != nil {
		return fmt.Errorf(`task: could not parse taskfile version "%s": %v`, e.Taskfile.Version, err)
	}

	if e.Context == nil {
		e.Context = context.Background()
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
	}
	switch {
	case version.IsV1(v):
		e.Compiler = &compilerv1.CompilerV1{
			Dir:    e.Dir,
			Vars:   e.taskvars,
			Logger: e.Logger,
		}
	case version.IsV2(v), version.IsV21(v):
		e.Compiler = &compilerv2.CompilerV2{
			Dir:          e.Dir,
			Taskvars:     e.taskvars,
			TaskfileVars: e.Taskfile.Vars,
			Expansions:   e.Taskfile.Expansions,
			Logger:       e.Logger,
		}
	case version.IsV22(v):
		return fmt.Errorf(`task: Taskfile versions greater than v2.1 not implemented in the version of Task`)
	}

	if !version.IsV21(v) && e.Taskfile.Output != "" {
		return fmt.Errorf(`task: Taskfile option "output" is only available starting on Taskfile version v2.1`)
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

	if !version.IsV21(v) {
		err := fmt.Errorf(`task: Taskfile option "ignore_error" is only available starting on Taskfile version v2.1`)

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

	e.taskCallCount = make(map[string]*int32, len(e.Taskfile.Tasks))
	for k := range e.Taskfile.Tasks {
		e.taskCallCount[k] = new(int32)
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

	if err := e.runDeps(ctx, t); err != nil {
		return err
	}

	if !e.Force {
		upToDate, err := isTaskUpToDate(ctx, t)
		if err != nil {
			return err
		}
		if upToDate {
			if !e.Silent {
				e.Logger.Errf(`task: Task "%s" is up to date`, t.Task)
			}
			return nil
		}
	}

	for i := range t.Cmds {
		if err := e.runCommand(ctx, t, call, i); err != nil {
			if err2 := statusOnError(t); err2 != nil {
				e.Logger.VerboseErrf("task: error cleaning status on error: %v", err2)
			}

			if _, ok := err.(interp.ExitCode); ok && t.IgnoreError {
				e.Logger.VerboseErrf("task: task error ignored: %v", err)
				continue
			}

			return &taskRunError{t.Task, err}
		}
	}
	return nil
}

func (e *Executor) runDeps(ctx context.Context, t *taskfile.Task) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, d := range t.Deps {
		d := d

		g.Go(func() error {
			return e.RunTask(ctx, taskfile.Call{Task: d.Task, Vars: d.Vars})
		})
	}

	return g.Wait()
}

func (e *Executor) runCommand(ctx context.Context, t *taskfile.Task, call taskfile.Call, i int) error {
	cmd := t.Cmds[i]

	switch {
	case cmd.Task != "":
		return e.RunTask(ctx, taskfile.Call{Task: cmd.Task, Vars: cmd.Vars})
	case cmd.Cmd != "":
		if e.Verbose || (!cmd.Silent && !t.Silent && !e.Silent) {
			e.Logger.Errf(cmd.Cmd)
		}

		if e.Dry {
			return nil
		}

		stdOut := e.Output.WrapWriter(e.Stdout, t.Prefix)
		stdErr := e.Output.WrapWriter(e.Stderr, t.Prefix)
		defer stdOut.Close()
		defer stdErr.Close()

		err := execext.RunCommand(&execext.RunCommandOptions{
			Context: ctx,
			Command: cmd.Cmd,
			Dir:     t.Dir,
			Env:     getEnviron(t),
			Stdin:   e.Stdin,
			Stdout:  stdOut,
			Stderr:  stdErr,
		})
		if _, ok := err.(interp.ExitCode); ok && cmd.IgnoreError {
			e.Logger.VerboseErrf("task: command error ignored: %v", err)
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

	envs := os.Environ()
	for k, v := range t.Env.ToStringMap() {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}
