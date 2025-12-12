package task

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sajari/fuzzy"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/taskfile/ast"
)

type (
	// An ExecutorOption is any type that can apply a configuration to an
	// [Executor].
	ExecutorOption interface {
		ApplyToExecutor(*Executor)
	}
	// An Executor is used for processing Taskfile(s) and executing the task(s)
	// within them.
	Executor struct {
		// Flags
		Dir                 string
		Entrypoint          string
		TempDir             TempDir
		Force               bool
		ForceAll            bool
		Insecure            bool
		Download            bool
		Offline             bool
		TrustedHosts        []string
		Timeout             time.Duration
		CacheExpiryDuration time.Duration
		RemoteCacheDir      string
		Watch               bool
		Verbose             bool
		Silent              bool
		DisableFuzzy        bool
		AssumeYes           bool
		AssumeTerm          bool // Used for testing
		Dry                 bool
		Summary             bool
		Parallel            bool
		Color               bool
		Concurrency         int
		Interval            time.Duration
		Failfast            bool

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

		fuzzyModel     *fuzzy.Model
		fuzzyModelOnce sync.Once

		concurrencySemaphore chan struct{}
		taskCallCount        map[string]*int32
		mkdirMutexMap        map[string]*sync.Mutex
		executionHashes      map[string]context.Context
		executionHashesMutex sync.Mutex
		watchedDirs          *xsync.Map[string, bool]
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
		opt.ApplyToExecutor(e)
	}
}

// WithDir sets the working directory of the [Executor]. By default, the
// directory is set to the user's current working directory.
func WithDir(dir string) ExecutorOption {
	return &dirOption{dir}
}

type dirOption struct {
	dir string
}

func (o *dirOption) ApplyToExecutor(e *Executor) {
	e.Dir = o.dir
}

// WithEntrypoint sets the entrypoint (main Taskfile) of the [Executor]. By
// default, Task will search for one of the default Taskfiles in the given
// directory.
func WithEntrypoint(entrypoint string) ExecutorOption {
	return &entrypointOption{entrypoint}
}

type entrypointOption struct {
	entrypoint string
}

func (o *entrypointOption) ApplyToExecutor(e *Executor) {
	e.Entrypoint = o.entrypoint
}

// WithTempDir sets the temporary directory that will be used by [Executor] for
// storing temporary files like checksums and cached remote files. By default,
// the temporary directory is set to the user's temporary directory.
func WithTempDir(tempDir TempDir) ExecutorOption {
	return &tempDirOption{tempDir}
}

type tempDirOption struct {
	tempDir TempDir
}

func (o *tempDirOption) ApplyToExecutor(e *Executor) {
	e.TempDir = o.tempDir
}

// WithForce ensures that the [Executor] always runs a task, even when
// fingerprinting or prompts would normally stop it.
func WithForce(force bool) ExecutorOption {
	return &forceOption{force}
}

type forceOption struct {
	force bool
}

func (o *forceOption) ApplyToExecutor(e *Executor) {
	e.Force = o.force
}

// WithForceAll ensures that the [Executor] always runs all tasks (including
// subtasks), even when fingerprinting or prompts would normally stop them.
func WithForceAll(forceAll bool) ExecutorOption {
	return &forceAllOption{forceAll}
}

type forceAllOption struct {
	forceAll bool
}

func (o *forceAllOption) ApplyToExecutor(e *Executor) {
	e.ForceAll = o.forceAll
}

// WithInsecure allows the [Executor] to make insecure connections when reading
// remote taskfiles. By default, insecure connections are rejected.
func WithInsecure(insecure bool) ExecutorOption {
	return &insecureOption{insecure}
}

type insecureOption struct {
	insecure bool
}

func (o *insecureOption) ApplyToExecutor(e *Executor) {
	e.Insecure = o.insecure
}

// WithDownload forces the [Executor] to download a fresh copy of the taskfile
// from the remote source.
func WithDownload(download bool) ExecutorOption {
	return &downloadOption{download}
}

type downloadOption struct {
	download bool
}

func (o *downloadOption) ApplyToExecutor(e *Executor) {
	e.Download = o.download
}

// WithOffline stops the [Executor] from being able to make network connections.
// It will still be able to read local files and cached copies of remote files.
func WithOffline(offline bool) ExecutorOption {
	return &offlineOption{offline}
}

type offlineOption struct {
	offline bool
}

func (o *offlineOption) ApplyToExecutor(e *Executor) {
	e.Offline = o.offline
}

// WithTrustedHosts configures the [Executor] with a list of trusted hosts for remote
// Taskfiles. Hosts in this list will not prompt for user confirmation.
func WithTrustedHosts(trustedHosts []string) ExecutorOption {
	return &trustedHostsOption{trustedHosts}
}

type trustedHostsOption struct {
	trustedHosts []string
}

func (o *trustedHostsOption) ApplyToExecutor(e *Executor) {
	e.TrustedHosts = o.trustedHosts
}

// WithTimeout sets the [Executor]'s timeout for fetching remote taskfiles. By
// default, the timeout is set to 10 seconds.
func WithTimeout(timeout time.Duration) ExecutorOption {
	return &timeoutOption{timeout}
}

type timeoutOption struct {
	timeout time.Duration
}

func (o *timeoutOption) ApplyToExecutor(e *Executor) {
	e.Timeout = o.timeout
}

// WithCacheExpiryDuration sets the duration after which the cache is considered
// expired. By default, the cache is 0 (disabled).
func WithCacheExpiryDuration(duration time.Duration) ExecutorOption {
	return &cacheExpiryDurationOption{duration: duration}
}

type cacheExpiryDurationOption struct {
	duration time.Duration
}

func (o *cacheExpiryDurationOption) ApplyToExecutor(r *Executor) {
	r.CacheExpiryDuration = o.duration
}

// WithRemoteCacheDir sets the directory where remote taskfiles are cached.
func WithRemoteCacheDir(dir string) ExecutorOption {
	return &remoteCacheDirOption{dir: dir}
}

type remoteCacheDirOption struct {
	dir string
}

func (o *remoteCacheDirOption) ApplyToExecutor(e *Executor) {
	e.RemoteCacheDir = o.dir
}

// WithWatch tells the [Executor] to keep running in the background and watch
// for changes to the fingerprint of the tasks that are run. When changes are
// detected, a new task run is triggered.
func WithWatch(watch bool) ExecutorOption {
	return &watchOption{watch}
}

type watchOption struct {
	watch bool
}

func (o *watchOption) ApplyToExecutor(e *Executor) {
	e.Watch = o.watch
}

// WithVerbose tells the [Executor] to output more information about the tasks
// that are run.
func WithVerbose(verbose bool) ExecutorOption {
	return &verboseOption{verbose}
}

type verboseOption struct {
	verbose bool
}

func (o *verboseOption) ApplyToExecutor(e *Executor) {
	e.Verbose = o.verbose
}

// WithSilent tells the [Executor] to suppress all output except for the output
// of the tasks that are run.
func WithSilent(silent bool) ExecutorOption {
	return &silentOption{silent}
}

type silentOption struct {
	silent bool
}

func (o *silentOption) ApplyToExecutor(e *Executor) {
	e.Silent = o.silent
}

// WithDisableFuzzy tells the [Executor] to disable fuzzy matching for task names.
func WithDisableFuzzy(disableFuzzy bool) ExecutorOption {
	return &disableFuzzyOption{disableFuzzy}
}

type disableFuzzyOption struct {
	disableFuzzy bool
}

func (o *disableFuzzyOption) ApplyToExecutor(e *Executor) {
	e.DisableFuzzy = o.disableFuzzy
}

// WithAssumeYes tells the [Executor] to assume "yes" for all prompts.
func WithAssumeYes(assumeYes bool) ExecutorOption {
	return &assumeYesOption{assumeYes}
}

type assumeYesOption struct {
	assumeYes bool
}

func (o *assumeYesOption) ApplyToExecutor(e *Executor) {
	e.AssumeYes = o.assumeYes
}

// WithAssumeTerm is used for testing purposes to simulate a terminal.
func WithAssumeTerm(assumeTerm bool) ExecutorOption {
	return &assumeTermOption{assumeTerm}
}

type assumeTermOption struct {
	assumeTerm bool
}

func (o *assumeTermOption) ApplyToExecutor(e *Executor) {
	e.AssumeTerm = o.assumeTerm
}

// WithDry tells the [Executor] to output the commands that would be run without
// actually running them.
func WithDry(dry bool) ExecutorOption {
	return &dryOption{dry}
}

type dryOption struct {
	dry bool
}

func (o *dryOption) ApplyToExecutor(e *Executor) {
	e.Dry = o.dry
}

// WithSummary tells the [Executor] to output a summary of the given tasks
// instead of running them.
func WithSummary(summary bool) ExecutorOption {
	return &summaryOption{summary}
}

type summaryOption struct {
	summary bool
}

func (o *summaryOption) ApplyToExecutor(e *Executor) {
	e.Summary = o.summary
}

// WithParallel tells the [Executor] to run tasks given in the same call in
// parallel.
func WithParallel(parallel bool) ExecutorOption {
	return &parallelOption{parallel}
}

type parallelOption struct {
	parallel bool
}

func (o *parallelOption) ApplyToExecutor(e *Executor) {
	e.Parallel = o.parallel
}

// WithColor tells the [Executor] whether or not to output using colorized
// strings.
func WithColor(color bool) ExecutorOption {
	return &colorOption{color}
}

type colorOption struct {
	color bool
}

func (o *colorOption) ApplyToExecutor(e *Executor) {
	e.Color = o.color
}

// WithConcurrency sets the maximum number of tasks that the [Executor] can run
// in parallel.
func WithConcurrency(concurrency int) ExecutorOption {
	return &concurrencyOption{concurrency}
}

type concurrencyOption struct {
	concurrency int
}

func (o *concurrencyOption) ApplyToExecutor(e *Executor) {
	e.Concurrency = o.concurrency
}

// WithInterval sets the interval at which the [Executor] will wait for
// duplicated events before running a task.
func WithInterval(interval time.Duration) ExecutorOption {
	return &intervalOption{interval}
}

type intervalOption struct {
	interval time.Duration
}

func (o *intervalOption) ApplyToExecutor(e *Executor) {
	e.Interval = o.interval
}

// WithOutputStyle sets the output style of the [Executor]. By default, the
// output style is set to the style defined in the Taskfile.
func WithOutputStyle(outputStyle ast.Output) ExecutorOption {
	return &outputStyleOption{outputStyle}
}

type outputStyleOption struct {
	outputStyle ast.Output
}

func (o *outputStyleOption) ApplyToExecutor(e *Executor) {
	e.OutputStyle = o.outputStyle
}

// WithTaskSorter sets the sorter that the [Executor] will use to sort tasks. By
// default, the sorter is set to sort tasks alphabetically, but with tasks with
// no namespace (in the root Taskfile) first.
func WithTaskSorter(sorter sort.Sorter) ExecutorOption {
	return &taskSorterOption{sorter}
}

type taskSorterOption struct {
	sorter sort.Sorter
}

func (o *taskSorterOption) ApplyToExecutor(e *Executor) {
	e.TaskSorter = o.sorter
}

// WithStdin sets the [Executor]'s standard input [io.Reader].
func WithStdin(stdin io.Reader) ExecutorOption {
	return &stdinOption{stdin}
}

type stdinOption struct {
	stdin io.Reader
}

func (o *stdinOption) ApplyToExecutor(e *Executor) {
	e.Stdin = o.stdin
}

// WithStdout sets the [Executor]'s standard output [io.Writer].
func WithStdout(stdout io.Writer) ExecutorOption {
	return &stdoutOption{stdout}
}

type stdoutOption struct {
	stdout io.Writer
}

func (o *stdoutOption) ApplyToExecutor(e *Executor) {
	e.Stdout = o.stdout
}

// WithStderr sets the [Executor]'s standard error [io.Writer].
func WithStderr(stderr io.Writer) ExecutorOption {
	return &stderrOption{stderr}
}

type stderrOption struct {
	stderr io.Writer
}

func (o *stderrOption) ApplyToExecutor(e *Executor) {
	e.Stderr = o.stderr
}

// WithIO sets the [Executor]'s standard input, output, and error to the same
// [io.ReadWriter].
func WithIO(rw io.ReadWriter) ExecutorOption {
	return &ioOption{rw}
}

type ioOption struct {
	rw io.ReadWriter
}

func (o *ioOption) ApplyToExecutor(e *Executor) {
	e.Stdin = o.rw
	e.Stdout = o.rw
	e.Stderr = o.rw
}

// WithVersionCheck tells the [Executor] whether or not to check the version of
func WithVersionCheck(enableVersionCheck bool) ExecutorOption {
	return &versionCheckOption{enableVersionCheck}
}

type versionCheckOption struct {
	enableVersionCheck bool
}

func (o *versionCheckOption) ApplyToExecutor(e *Executor) {
	e.EnableVersionCheck = o.enableVersionCheck
}

// WithFailfast tells the [Executor] whether or not to check the version of
func WithFailfast(failfast bool) ExecutorOption {
	return &failfastOption{failfast}
}

type failfastOption struct {
	failfast bool
}

func (o *failfastOption) ApplyToExecutor(e *Executor) {
	e.Failfast = o.failfast
}
