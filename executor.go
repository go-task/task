package task

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sajari/fuzzy"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/taskfile/ast"
)

type (
	// An ExecutorOption is a functional option for an [Executor].
	ExecutorOption func(*Executor)
	// An Executor is used for processing Taskfile(s) and executing the task(s)
	// within them.
	Executor struct {
		// Flags
		Dir         string
		Entrypoint  string
		TempDir     TempDir
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

		// I/O
		Stdin  io.Reader
		Stdout io.Writer
		Stderr io.Writer

		// Internal
		Taskfile           *ast.Taskfile
		Logger             *logger.Logger
		Compiler           *Compiler
		Output             output.Output
		OutputStyle        ast.Output
		TaskSorter         sort.Sorter
		UserWorkingDir     string
		EnableVersionCheck bool

		fuzzyModel *fuzzy.Model

		concurrencySemaphore chan struct{}
		taskCallCount        map[string]*int32
		mkdirMutexMap        map[string]*sync.Mutex
		executionHashes      map[string]context.Context
		executionHashesMutex sync.Mutex
	}
	TempDir struct {
		Remote      string
		Fingerprint string
	}
)

// NewExecutor creates a new [Executor] and applies the given functional options
// to it.
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		Timeout:              time.Second * 10,
		Interval:             time.Second * 5,
		Stdin:                os.Stdin,
		Stdout:               os.Stdout,
		Stderr:               os.Stderr,
		Logger:               nil,
		Compiler:             nil,
		Output:               nil,
		OutputStyle:          ast.Output{},
		TaskSorter:           sort.AlphaNumericWithRootTasksFirst,
		UserWorkingDir:       "",
		fuzzyModel:           nil,
		concurrencySemaphore: nil,
		taskCallCount:        map[string]*int32{},
		mkdirMutexMap:        map[string]*sync.Mutex{},
		executionHashes:      map[string]context.Context{},
		executionHashesMutex: sync.Mutex{},
	}
	e.Options(opts...)
	return e
}

// Options loops through the given [ExecutorOption] functions and applies them
// to the [Executor].
func (e *Executor) Options(opts ...ExecutorOption) {
	for _, opt := range opts {
		opt(e)
	}
}

// ExecutorWithDir sets the working directory of the [Executor]. By default, the
// directory is set to the user's current working directory.
func ExecutorWithDir(dir string) ExecutorOption {
	return func(e *Executor) {
		e.Dir = dir
	}
}

// ExecutorWithEntrypoint sets the entrypoint (main Taskfile) of the [Executor].
// By default, Task will search for one of the default Taskfiles in the given
// directory.
func ExecutorWithEntrypoint(entrypoint string) ExecutorOption {
	return func(e *Executor) {
		e.Entrypoint = entrypoint
	}
}

// ExecutorWithTempDir sets the temporary directory that will be used by
// [Executor] for storing temporary files like checksums and cached remote
// files. By default, the temporary directory is set to the user's temporary
// directory.
func ExecutorWithTempDir(tempDir TempDir) ExecutorOption {
	return func(e *Executor) {
		e.TempDir = tempDir
	}
}

// ExecutorWithForce ensures that the [Executor] always runs a task, even when
// fingerprinting or prompts would normally stop it.
func ExecutorWithForce(force bool) ExecutorOption {
	return func(e *Executor) {
		e.Force = force
	}
}

// ExecutorWithForceAll ensures that the [Executor] always runs all tasks
// (including subtasks), even when fingerprinting or prompts would normally stop
// them.
func ExecutorWithForceAll(forceAll bool) ExecutorOption {
	return func(e *Executor) {
		e.ForceAll = forceAll
	}
}

// ExecutorWithInsecure allows the [Executor] to make insecure connections when
// reading remote taskfiles. By default, insecure connections are rejected.
func ExecutorWithInsecure(insecure bool) ExecutorOption {
	return func(e *Executor) {
		e.Insecure = insecure
	}
}

// ExecutorWithDownload forces the [Executor] to download a fresh copy of the
// taskfile from the remote source.
func ExecutorWithDownload(download bool) ExecutorOption {
	return func(e *Executor) {
		e.Download = download
	}
}

// ExecutorWithOffline stops the [Executor] from being able to make network
// connections. It will still be able to read local files and cached copies of
// remote files.
func ExecutorWithOffline(offline bool) ExecutorOption {
	return func(e *Executor) {
		e.Offline = offline
	}
}

// ExecutorWithTimeout sets the [Executor]'s timeout for fetching remote
// taskfiles. By default, the timeout is set to 10 seconds.
func ExecutorWithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.Timeout = timeout
	}
}

// ExecutorWithWatch tells the [Executor] to keep running in the background and
// watch for changes to the fingerprint of the tasks that are run. When changes
// are detected, a new task run is triggered.
func ExecutorWithWatch(watch bool) ExecutorOption {
	return func(e *Executor) {
		e.Watch = watch
	}
}

// ExecutorWithVerbose tells the [Executor] to output more information about the
// tasks that are run.
func ExecutorWithVerbose(verbose bool) ExecutorOption {
	return func(e *Executor) {
		e.Verbose = verbose
	}
}

// ExecutorWithSilent tells the [Executor] to suppress all output except for the
// output of the tasks that are run.
func ExecutorWithSilent(silent bool) ExecutorOption {
	return func(e *Executor) {
		e.Silent = silent
	}
}

// ExecutorWithAssumeYes tells the [Executor] to assume "yes" for all prompts.
func ExecutorWithAssumeYes(assumeYes bool) ExecutorOption {
	return func(e *Executor) {
		e.AssumeYes = assumeYes
	}
}

// WithAssumeTerm is used for testing purposes to simulate a terminal.
func ExecutorWithAssumeTerm(assumeTerm bool) ExecutorOption {
	return func(e *Executor) {
		e.AssumeTerm = assumeTerm
	}
}

// ExecutorWithDry tells the [Executor] to output the commands that would be run
// without actually running them.
func ExecutorWithDry(dry bool) ExecutorOption {
	return func(e *Executor) {
		e.Dry = dry
	}
}

// ExecutorWithSummary tells the [Executor] to output a summary of the given
// tasks instead of running them.
func ExecutorWithSummary(summary bool) ExecutorOption {
	return func(e *Executor) {
		e.Summary = summary
	}
}

// ExecutorWithParallel tells the [Executor] to run tasks given in the same call
// in parallel.
func ExecutorWithParallel(parallel bool) ExecutorOption {
	return func(e *Executor) {
		e.Parallel = parallel
	}
}

// ExecutorWithColor tells the [Executor] whether or not to output using
// colorized strings.
func ExecutorWithColor(color bool) ExecutorOption {
	return func(e *Executor) {
		e.Color = color
	}
}

// ExecutorWithConcurrency sets the maximum number of tasks that the [Executor]
// can run in parallel.
func ExecutorWithConcurrency(concurrency int) ExecutorOption {
	return func(e *Executor) {
		e.Concurrency = concurrency
	}
}

// ExecutorWithInterval sets the interval at which the [Executor] will check for
// changes when watching tasks.
func ExecutorWithInterval(interval time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.Interval = interval
	}
}

// ExecutorWithOutputStyle sets the output style of the [Executor]. By default,
// the output style is set to the style defined in the Taskfile.
func ExecutorWithOutputStyle(outputStyle ast.Output) ExecutorOption {
	return func(e *Executor) {
		e.OutputStyle = outputStyle
	}
}

// ExecutorWithTaskSorter sets the sorter that the [Executor] will use to sort
// tasks. By default, the sorter is set to sort tasks alphabetically, but with
// tasks with no namespace (in the root Taskfile) first.
func ExecutorWithTaskSorter(sorter sort.Sorter) ExecutorOption {
	return func(e *Executor) {
		e.TaskSorter = sorter
	}
}

// ExecutorWithStdin sets the [Executor]'s standard input [io.Reader].
func ExecutorWithStdin(stdin io.Reader) ExecutorOption {
	return func(e *Executor) {
		e.Stdin = stdin
	}
}

// ExecutorWithStdout sets the [Executor]'s standard output [io.Writer].
func ExecutorWithStdout(stdout io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.Stdout = stdout
	}
}

// ExecutorWithStderr sets the [Executor]'s standard error [io.Writer].
func ExecutorWithStderr(stderr io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.Stderr = stderr
	}
}

// ExecutorWithIO sets the [Executor]'s standard input, output, and error to the
// same [io.ReadWriter].
func ExecutorWithIO(rw io.ReadWriter) ExecutorOption {
	return func(e *Executor) {
		e.Stdin = rw
		e.Stdout = rw
		e.Stderr = rw
	}
}

// ExecutorWithVersionCheck tells the [Executor] whether or not to check the
// version of
func ExecutorWithVersionCheck(enableVersionCheck bool) ExecutorOption {
	return func(e *Executor) {
		e.EnableVersionCheck = enableVersionCheck
	}
}
