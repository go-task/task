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

// WithDir sets the working directory of the [Executor]. By default, the
// directory is set to the user's current working directory.
func WithDir(dir string) ExecutorOption {
	return func(e *Executor) {
		e.Dir = dir
	}
}

// WithEntrypoint sets the entrypoint (main Taskfile) of the [Executor]. By
// default, Task will search for one of the default Taskfiles in the given
// directory.
func WithEntrypoint(entrypoint string) ExecutorOption {
	return func(e *Executor) {
		e.Entrypoint = entrypoint
	}
}

// WithTempDir sets the temporary directory that will be used by [Executor] for
// storing temporary files like checksums and cached remote files. By default,
// the temporary directory is set to the user's temporary directory.
func WithTempDir(tempDir TempDir) ExecutorOption {
	return func(e *Executor) {
		e.TempDir = tempDir
	}
}

// WithForce ensures that the [Executor] always runs a task, even when
// fingerprinting or prompts would normally stop it.
func WithForce(force bool) ExecutorOption {
	return func(e *Executor) {
		e.Force = force
	}
}

// WithForceAll ensures that the [Executor] always runs all tasks (including
// subtasks), even when fingerprinting or prompts would normally stop them.
func WithForceAll(forceAll bool) ExecutorOption {
	return func(e *Executor) {
		e.ForceAll = forceAll
	}
}

// WithInsecure allows the [Executor] to make insecure connections when reading
// remote taskfiles. By default, insecure connections are rejected.
func WithInsecure(insecure bool) ExecutorOption {
	return func(e *Executor) {
		e.Insecure = insecure
	}
}

// WithDownload forces the [Executor] to download a fresh copy of the taskfile
// from the remote source.
func WithDownload(download bool) ExecutorOption {
	return func(e *Executor) {
		e.Download = download
	}
}

// WithOffline stops the [Executor] from being able to make network connections.
// It will still be able to read local files and cached copies of remote files.
func WithOffline(offline bool) ExecutorOption {
	return func(e *Executor) {
		e.Offline = offline
	}
}

// WithTimeout sets the [Executor]'s timeout for fetching remote taskfiles. By
// default, the timeout is set to 10 seconds.
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.Timeout = timeout
	}
}

// WithWatch tells the [Executor] to keep running in the background and watch
// for changes to the fingerprint of the tasks that are run. When changes are
// detected, a new task run is triggered.
func WithWatch(watch bool) ExecutorOption {
	return func(e *Executor) {
		e.Watch = watch
	}
}

// WithVerbose tells the [Executor] to output more information about the tasks
// that are run.
func WithVerbose(verbose bool) ExecutorOption {
	return func(e *Executor) {
		e.Verbose = verbose
	}
}

// WithSilent tells the [Executor] to suppress all output except for the output
// of the tasks that are run.
func WithSilent(silent bool) ExecutorOption {
	return func(e *Executor) {
		e.Silent = silent
	}
}

// WithAssumeYes tells the [Executor] to assume "yes" for all prompts.
func WithAssumeYes(assumeYes bool) ExecutorOption {
	return func(e *Executor) {
		e.AssumeYes = assumeYes
	}
}

// WithAssumeTerm is used for testing purposes to simulate a terminal.
func WithDry(dry bool) ExecutorOption {
	return func(e *Executor) {
		e.Dry = dry
	}
}

// WithSummary tells the [Executor] to output a summary of the given tasks
// instead of running them.
func WithSummary(summary bool) ExecutorOption {
	return func(e *Executor) {
		e.Summary = summary
	}
}

// WithParallel tells the [Executor] to run tasks given in the same call
// in parallel.
func WithParallel(parallel bool) ExecutorOption {
	return func(e *Executor) {
		e.Parallel = parallel
	}
}

// WithColor tells the [Executor] whether or not to output using colorized
// strings.
func WithColor(color bool) ExecutorOption {
	return func(e *Executor) {
		e.Color = color
	}
}

// WithConcurrency sets the maximum number of tasks that the [Executor] can run
// in parallel.
func WithConcurrency(concurrency int) ExecutorOption {
	return func(e *Executor) {
		e.Concurrency = concurrency
	}
}

// WithInterval sets the interval at which the [Executor] will check for changes
// when watching tasks.
func WithInterval(interval time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.Interval = interval
	}
}

// WithOutputStyle sets the output style of the [Executor]. By default, the
// output style is set to the style defined in the Taskfile.
func WithOutputStyle(outputStyle ast.Output) ExecutorOption {
	return func(e *Executor) {
		e.OutputStyle = outputStyle
	}
}

// WithTaskSorter sets the sorter that the [Executor] will use to sort tasks.
// By default, the sorter is set to sort tasks alphabetically, but with tasks
// with no namespace (in the root Taskfile) first.
func WithTaskSorter(sorter sort.Sorter) ExecutorOption {
	return func(e *Executor) {
		e.TaskSorter = sorter
	}
}

// WithStdin sets the [Executor]'s standard input [io.Reader].
func WithStdin(stdin io.Reader) ExecutorOption {
	return func(e *Executor) {
		e.Stdin = stdin
	}
}

// WithStdout sets the [Executor]'s standard output [io.Writer].
func WithStdout(stdout io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.Stdout = stdout
	}
}

// WithStderr sets the [Executor]'s standard error [io.Writer].
func WithStderr(stderr io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.Stderr = stderr
	}
}

// WithIO sets the [Executor]'s standard input, output, and error to the same
// [io.ReadWriter].
func WithIO(rw io.ReadWriter) ExecutorOption {
	return func(e *Executor) {
		e.Stdin = rw
		e.Stdout = rw
		e.Stderr = rw
	}
}

// WithVersionCheck tells the [Executor] whether or not to check the version of
func WithVersionCheck(enableVersionCheck bool) ExecutorOption {
	return func(e *Executor) {
		e.EnableVersionCheck = enableVersionCheck
	}
}
