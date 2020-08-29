package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
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

	// MaximumDepLevel defines the maximum depth of the dependency analysis.
	// The depth is 2 if a (0) depends on b (1) depends on c (2).
	// We use this to prevent the dependency analysis to end up in an infinite loop.
	MaximumDepLevel = 50
)

// Executor executes a Taskfile
type Executor struct {
	Taskfile *taskfile.Taskfile

	Dir        string
	Entrypoint string
	Force      bool
	Watch      bool
	Verbose    bool
	Silent     bool
	Dry        bool
	Summary    bool
	Parallel   bool
	Color      bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Logger      *logger.Logger
	Compiler    compiler.Compiler
	Output      output.Output
	OutputStyle string

	taskvars *taskfile.Vars

	callStackMux  *sync.Mutex
	taskCallCount map[string]*int32
	mkdirMutexMap map[string]*sync.Mutex
	depMutexMap   map[string]*sync.Mutex
}

// Run runs Task
func (e *Executor) Run(ctx context.Context, calls ...taskfile.Call) error {
	// check if given tasks exist
	for _, c := range calls {
		if _, ok := e.Taskfile.Tasks[c.Task]; !ok {
			// FIXME: move to the main package
			e.PrintTasksHelp()
			return &taskNotFoundError{taskName: c.Task}
		}
	}

	if e.Summary {
		for i, c := range calls {
			compiledTask, err := e.CompiledTask(c)
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
	if e.Entrypoint == "" {
		e.Entrypoint = "Taskfile.yml"
	}

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

	if v > 3.0 {
		return fmt.Errorf(`task: Taskfile versions greater than v3.0 not implemented in the version of Task`)
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
			TaskfileVars: e.Taskfile.Vars,
			Logger:       e.Logger,
		}
	}

	if v < 2.1 && e.Taskfile.Output != "" {
		return fmt.Errorf(`task: Taskfile option "output" is only available starting on Taskfile version v2.1`)
	}
	if v < 2.2 && len(e.Taskfile.Includes) > 0 {
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
		for _, taskfile := range e.Taskfile.Includes {
			if taskfile.AdvancedImport {
				return errors.New(`task: Import with additional parameters is only available starting on Taskfile version v3`)
			}
		}
	}

	e.callStackMux = &sync.Mutex{}
	e.taskCallCount = make(map[string]*int32, len(e.Taskfile.Tasks))
	e.mkdirMutexMap = make(map[string]*sync.Mutex, len(e.Taskfile.Tasks))
	e.depMutexMap = make(map[string]*sync.Mutex, len(e.Taskfile.Tasks))
	for k := range e.Taskfile.Tasks {
		e.taskCallCount[k] = new(int32)
		e.mkdirMutexMap[k] = &sync.Mutex{}
		e.depMutexMap[k] = &sync.Mutex{}
	}
	return nil
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call taskfile.Call) error {
	return e.runTask(ctx, make(taskfile.CallStack, 0, 16), call, true)
}

// runTask runs a task by its name
func (e *Executor) runTask(ctx context.Context, callStack taskfile.CallStack, call taskfile.Call, runDeps bool) error {
	if callStack.Contains(call) {
		return InfiniteCallLoopError{causeTask: call.Task, callStack: callStack}
	}

	callStack = callStack.Add(call)

	t, err := e.CompiledTask(call)
	if err != nil {
		return err
	}
	if !e.Watch && atomic.AddInt32(e.taskCallCount[call.Task], 1) >= MaximumTaskCall {
		return &MaximumTaskCallExceededError{task: call.Task}
	}

	if runDeps {
		if err := e.runDeps(ctx, callStack, t); err != nil {
			return err
		}
	}

	if !e.Force {
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
		if err := e.runCommand(ctx, callStack, t, i); err != nil {
			if err2 := e.statusOnError(t); err2 != nil {
				e.Logger.VerboseErrf(logger.Yellow, "task: error cleaning status on error: %v", err2)
			}

			if execext.IsExitError(err) && t.IgnoreError {
				e.Logger.VerboseErrf(logger.Yellow, "task: task error ignored: %v", err)
				continue
			}

			return &RunError{t.Task, err}
		}
	}
	return nil
}

// collectDeps collects the dependencies of the given task.
// It returns the dependencies by level/depth.
//
// Example:
// a depends on b and c
// b depends on c
// c depends on d
// d depends on nothing
//
// When getting the dependencies of a, the following is returned:
// [0]{"b","c"} - because a depends on b and c
// [1]{"c","d"} - because b (from level 0) depends on c, and c (from level 0) depends on d
// [2]{"d"} - because c (from level 1) depends on d
//
// It could thus be that one task is returned more than once - if multiple tasks depend on the same task,
// but in different levels.
func (e *Executor) collectDeps(t *taskfile.Task) (map[int][]*taskfile.Dep, error) {
	collection := make(map[int][]*taskfile.Dep, 1)
	err := e.recursivelyCollectDeps(t, 0, collection)
	if err != nil {
		return nil, err
	}
	return collection, err
}

// recursivelyCollectDeps recursively collects deps of the given task. It groups them by level.
func (e *Executor) recursivelyCollectDeps(
	t *taskfile.Task,
	level int,
	collection map[int][]*taskfile.Dep,
) error {
	if level == MaximumDepLevel {
		return MaxDepLevelReachedError{level}
	}

	collection[level] = append(collection[level], t.Deps...)

	for _, d := range t.Deps {
		depTask, err := e.CompiledTask(d.ToCall())
		if err != nil {
			return err
		}
		if len(depTask.Deps) == 0 {
			continue
		}

		// If one of this dependency's dependencies depends on the input task,
		// we have a dependency cycle.
		for _, depDep := range depTask.Deps {
			if depDep.Task == t.Task {
				return DirectDepCycleError{depDep.Task, d.Task}
			}
		}

		err = e.recursivelyCollectDeps(depTask, level+1, collection)
		if err != nil {
			return err
		}
	}

	return nil
}

// runDeps runs the dependencies of the given task.
func (e *Executor) runDeps(ctx context.Context, callStack taskfile.CallStack, t *taskfile.Task) error {
	deps, err := e.collectDeps(t)
	if err != nil {
		return err
	}

	// Keep track of which deps we've run so that we can prevent a dep
	// from being run again if it has already ran on a deeper level.
	ranDeps := make([]*taskfile.Dep, 0, len(deps))

	// Start at the deepest level and run all deps there,
	// then move up levels until all deps have been run.
	for level := len(deps) - 1; level >= 0; level-- {
		g, ctx := errgroup.WithContext(ctx)

	depLoop:
		for _, d := range deps[level] {
			d := d

			// If this dep was already run, don't run it again.
			for _, ranDep := range ranDeps {
				if d.Task == ranDep.Task && reflect.DeepEqual(d.Vars, ranDep.Vars) {
					continue depLoop
				}
			}

			g.Go(func() error {
				return e.runDep(ctx, callStack, d)
			})

			ranDeps = append(ranDeps, d)
		}

		err := g.Wait()
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Executor) runDep(ctx context.Context, callStack taskfile.CallStack, d *taskfile.Dep) error {
	// Even though in runTask we check if we are not ending up in an endless loop,
	// we must do it here, too because it may prevent a deadlock.
	// Example: a has dep b which runs a (directly, no dep).
	// When a runs, the dependencies are run. Only one dependency is allowed to run at a time,
	// so when dep b runs, a cannot finish until dep b finishes.
	// B runs a however (directly), and a must run dep b, which is stuck because the first run of
	// a is still waiting for it.
	if callStack.Contains(d.ToCall()) {
		return InfiniteCallLoopError{causeTask: d.Task, callStack: callStack}
	}

	e.depMutexMap[d.Task].Lock()
	defer e.depMutexMap[d.Task].Unlock()
	return e.runTask(ctx, callStack, d.ToCall(), false)
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

func (e *Executor) runCommand(ctx context.Context, callStack taskfile.CallStack, t *taskfile.Task, i int) error {
	cmd := t.Cmds[i]

	switch {
	case cmd.Task != "":
		err := e.runTask(ctx, callStack, cmd.ToCall(), true)
		if err != nil {
			return err
		}
		return nil
	case cmd.Cmd != "":
		if e.Verbose || (!cmd.Silent && !t.Silent && !e.Taskfile.Silent && !e.Silent) {
			e.Logger.Errf(logger.Green, "task: %s", cmd.Cmd)
		}

		if e.Dry {
			return nil
		}

		stdOut := e.Output.WrapWriter(e.Stdout, t.Prefix)
		stdErr := e.Output.WrapWriter(e.Stderr, t.Prefix)
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
			e.Logger.VerboseErrf(logger.Yellow, "task: command error ignored: %v", err)
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
		if s, ok := v.(string); ok {
			environ = append(environ, fmt.Sprintf("%s=%s", k, s))
		}
	}
	return environ
}
