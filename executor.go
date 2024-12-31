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
	// An ExecutorOption is a functional option for an Executor
	ExecutorOption func(*Executor)
	// An Executor is a type that is used for processing and executing Taskfiles
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

func (e *Executor) Options(opts ...ExecutorOption) {
	for _, opt := range opts {
		opt(e)
	}
}

func WithDir(dir string) ExecutorOption {
	return func(e *Executor) {
		e.Dir = dir
	}
}

func WithEntrypoint(entrypoint string) ExecutorOption {
	return func(e *Executor) {
		e.Entrypoint = entrypoint
	}
}

func WithTempDir(tempDir TempDir) ExecutorOption {
	return func(e *Executor) {
		e.TempDir = tempDir
	}
}

func WithForce(force bool) ExecutorOption {
	return func(e *Executor) {
		e.Force = force
	}
}

func WithForceAll(forceAll bool) ExecutorOption {
	return func(e *Executor) {
		e.ForceAll = forceAll
	}
}

func WithInsecure(insecure bool) ExecutorOption {
	return func(e *Executor) {
		e.Insecure = insecure
	}
}

func WithDownload(download bool) ExecutorOption {
	return func(e *Executor) {
		e.Download = download
	}
}

func WithOffline(offline bool) ExecutorOption {
	return func(e *Executor) {
		e.Offline = offline
	}
}

func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.Timeout = timeout
	}
}

func WithWatch(watch bool) ExecutorOption {
	return func(e *Executor) {
		e.Watch = watch
	}
}

func WithVerbose(verbose bool) ExecutorOption {
	return func(e *Executor) {
		e.Verbose = verbose
	}
}

func WithSilent(silent bool) ExecutorOption {
	return func(e *Executor) {
		e.Silent = silent
	}
}

func WithAssumeYes(assumeYes bool) ExecutorOption {
	return func(e *Executor) {
		e.AssumeYes = assumeYes
	}
}

func WithDry(dry bool) ExecutorOption {
	return func(e *Executor) {
		e.Dry = dry
	}
}

func WithSummary(summary bool) ExecutorOption {
	return func(e *Executor) {
		e.Summary = summary
	}
}

func WithParallel(parallel bool) ExecutorOption {
	return func(e *Executor) {
		e.Parallel = parallel
	}
}

func WithColor(color bool) ExecutorOption {
	return func(e *Executor) {
		e.Color = color
	}
}

func WithConcurrency(concurrency int) ExecutorOption {
	return func(e *Executor) {
		e.Concurrency = concurrency
	}
}

func WithInterval(interval time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.Interval = interval
	}
}

func WithOutputStyle(outputStyle ast.Output) ExecutorOption {
	return func(e *Executor) {
		e.OutputStyle = outputStyle
	}
}

func WithTaskSorter(sorter sort.Sorter) ExecutorOption {
	return func(e *Executor) {
		e.TaskSorter = sorter
	}
}

func WithStdin(stdin io.Reader) ExecutorOption {
	return func(e *Executor) {
		e.Stdin = stdin
	}
}

func WithStdout(stdout io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.Stdout = stdout
	}
}

func WithStderr(stderr io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.Stderr = stderr
	}
}

func WithVersionCheck(enableVersionCheck bool) ExecutorOption {
	return func(e *Executor) {
		e.EnableVersionCheck = enableVersionCheck
	}
}
