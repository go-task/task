package flags

import (
	"cmp"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/taskfile/ast"
	"github.com/go-task/task/v3/taskrc"
	taskrcast "github.com/go-task/task/v3/taskrc/ast"
)

const usage = `Usage: task [flags...] [task...]

Runs the specified task(s). Falls back to the "default" task if no task name
was specified, or lists all tasks if an unknown task name was specified.

Example: 'task hello' with the following 'Taskfile.yml' file will generate an
'output.txt' file with the content "hello".

'''
version: '3'
tasks:
  hello:
    cmds:
      - echo "I am going to write a file named 'output.txt' now."
      - echo "hello" > output.txt
    generates:
      - output.txt
'''

Options:
`

var (
	Version             bool
	Help                bool
	Init                bool
	Completion          string
	List                bool
	ListAll             bool
	ListJson            bool
	TaskSort            string
	Status              bool
	NoStatus            bool
	Nested              bool
	Insecure            bool
	Force               bool
	ForceAll            bool
	Watch               bool
	Verbose             bool
	Silent              bool
	DisableFuzzy        bool
	AssumeYes           bool
	Dry                 bool
	Summary             bool
	ExitCode            bool
	Parallel            bool
	Concurrency         int
	Dir                 string
	Entrypoint          string
	Output              ast.Output
	Color               bool
	Interval            time.Duration
	Failfast            bool
	Global              bool
	Experiments         bool
	Download            bool
	Offline             bool
	TrustedHosts        []string
	ClearCache          bool
	Timeout             time.Duration
	CacheExpiryDuration time.Duration
	RemoteCacheDir      string
)

func init() {
	// Config files can enable experiments which alter the availability and/or
	// behavior of some flags, so we need to parse the experiments before the
	// flags. However, we need the --taskfile and --dir flags before we can
	// parse the experiments as they can alter the location of the config files.
	// Because of this circular dependency, we parse the flags twice. First, we
	// get the --taskfile and --dir flags, then we parse the experiments, then
	// we parse the flags again to get the full set. We use a flagset here so
	// that we can parse a subset of flags without exiting on error.
	var dir, entrypoint string
	fs := pflag.NewFlagSet("experiments", pflag.ContinueOnError)
	fs.StringVarP(&dir, "dir", "d", "", "")
	fs.StringVarP(&entrypoint, "taskfile", "t", "", "")
	fs.Usage = func() {}
	_ = fs.Parse(os.Args[1:])

	// Parse the experiments
	dir = cmp.Or(dir, filepath.Dir(entrypoint))

	config, _ := taskrc.GetConfig(dir)
	experiments.ParseWithConfig(dir, config)

	// Parse the rest of the flags
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	pflag.Usage = func() {
		log.Print(usage)
		pflag.PrintDefaults()
	}

	pflag.BoolVar(&Version, "version", false, "Show Task version.")
	pflag.BoolVarP(&Help, "help", "h", false, "Shows Task usage.")
	pflag.BoolVarP(&Init, "init", "i", false, "Creates a new Taskfile.yml in the current folder.")
	pflag.StringVar(&Completion, "completion", "", "Generates shell completion script.")
	pflag.BoolVarP(&List, "list", "l", false, "Lists tasks with description of current Taskfile.")
	pflag.BoolVarP(&ListAll, "list-all", "a", false, "Lists tasks with or without a description.")
	pflag.BoolVarP(&ListJson, "json", "j", false, "Formats task list as JSON.")
	pflag.StringVar(&TaskSort, "sort", "", "Changes the order of the tasks when listed. [default|alphanumeric|none].")
	pflag.BoolVar(&Status, "status", false, "Exits with non-zero exit code if any of the given tasks is not up-to-date.")
	pflag.BoolVar(&NoStatus, "no-status", false, "Ignore status when listing tasks as JSON")
	pflag.BoolVar(&Nested, "nested", false, "Nest namespaces when listing tasks as JSON")
	pflag.BoolVar(&Insecure, "insecure", getConfig(config, func() *bool { return config.Remote.Insecure }, false), "Forces Task to download Taskfiles over insecure connections.")
	pflag.BoolVarP(&Watch, "watch", "w", false, "Enables watch of the given task.")
	pflag.BoolVarP(&Verbose, "verbose", "v", getConfig(config, func() *bool { return config.Verbose }, false), "Enables verbose mode.")
	pflag.BoolVarP(&Silent, "silent", "s", false, "Disables echoing.")
	pflag.BoolVar(&DisableFuzzy, "disable-fuzzy", getConfig(config, func() *bool { return config.DisableFuzzy }, false), "Disables fuzzy matching for task names.")
	pflag.BoolVarP(&AssumeYes, "yes", "y", false, "Assume \"yes\" as answer to all prompts.")
	pflag.BoolVarP(&Parallel, "parallel", "p", false, "Executes tasks provided on command line in parallel.")
	pflag.BoolVarP(&Dry, "dry", "n", false, "Compiles and prints tasks in the order that they would be run, without executing them.")
	pflag.BoolVar(&Summary, "summary", false, "Show summary about a task.")
	pflag.BoolVarP(&ExitCode, "exit-code", "x", false, "Pass-through the exit code of the task command.")
	pflag.StringVarP(&Dir, "dir", "d", "", "Sets the directory in which Task will execute and look for a Taskfile.")
	pflag.StringVarP(&Entrypoint, "taskfile", "t", "", `Choose which Taskfile to run. Defaults to "Taskfile.yml".`)
	pflag.StringVarP(&Output.Name, "output", "o", "", "Sets output style: [interleaved|group|prefixed].")
	pflag.StringVar(&Output.Group.Begin, "output-group-begin", "", "Message template to print before a task's grouped output.")
	pflag.StringVar(&Output.Group.End, "output-group-end", "", "Message template to print after a task's grouped output.")
	pflag.BoolVar(&Output.Group.ErrorOnly, "output-group-error-only", false, "Swallow output from successful tasks.")
	pflag.BoolVarP(&Color, "color", "c", getConfig(config, func() *bool { return config.Color }, true), "Colored output. Enabled by default. Set flag to false or use NO_COLOR=1 to disable.")
	pflag.IntVarP(&Concurrency, "concurrency", "C", getConfig(config, func() *int { return config.Concurrency }, 0), "Limit number of tasks to run concurrently.")
	pflag.DurationVarP(&Interval, "interval", "I", 0, "Interval to watch for changes.")
	pflag.BoolVarP(&Failfast, "failfast", "F", getConfig(config, func() *bool { return &config.Failfast }, false), "When running tasks in parallel, stop all tasks if one fails.")
	pflag.BoolVarP(&Global, "global", "g", false, "Runs global Taskfile, from $HOME/{T,t}askfile.{yml,yaml}.")
	pflag.BoolVar(&Experiments, "experiments", false, "Lists all the available experiments and whether or not they are enabled.")

	// Gentle force experiment will override the force flag and add a new force-all flag
	if experiments.GentleForce.Enabled() {
		pflag.BoolVarP(&Force, "force", "f", false, "Forces execution of the directly called task.")
		pflag.BoolVar(&ForceAll, "force-all", false, "Forces execution of the called task and all its dependant tasks.")
	} else {
		pflag.BoolVarP(&ForceAll, "force", "f", false, "Forces execution even when the task is up-to-date.")
	}

	// Remote Taskfiles experiment will adds the "download" and "offline" flags
	if experiments.RemoteTaskfiles.Enabled() {
		pflag.BoolVar(&Download, "download", false, "Downloads a cached version of a remote Taskfile.")
		pflag.BoolVar(&Offline, "offline", getConfig(config, func() *bool { return config.Remote.Offline }, false), "Forces Task to only use local or cached Taskfiles.")
		pflag.StringSliceVar(&TrustedHosts, "trusted-hosts", getConfig(config, func() *[]string { return &config.Remote.TrustedHosts }, nil), "List of trusted hosts for remote Taskfiles (comma-separated).")
		pflag.DurationVar(&Timeout, "timeout", getConfig(config, func() *time.Duration { return config.Remote.Timeout }, time.Second*10), "Timeout for downloading remote Taskfiles.")
		pflag.BoolVar(&ClearCache, "clear-cache", false, "Clear the remote cache.")
		pflag.DurationVar(&CacheExpiryDuration, "expiry", getConfig(config, func() *time.Duration { return config.Remote.CacheExpiry }, 0), "Expiry duration for cached remote Taskfiles.")
		pflag.StringVar(&RemoteCacheDir, "remote-cache-dir", getConfig(config, func() *string { return config.Remote.CacheDir }, env.GetTaskEnv("REMOTE_DIR")), "Directory to cache remote Taskfiles.")
	}
	pflag.Parse()

	// Auto-detect color based on environment when not explicitly configured
	// Priority: CLI flag > taskrc config > NO_COLOR > FORCE_COLOR/CI > default
	colorExplicitlySet := pflag.Lookup("color").Changed || (config != nil && config.Color != nil)
	if !colorExplicitlySet {
		if os.Getenv("NO_COLOR") != "" {
			Color = false
			color.NoColor = true
		} else if os.Getenv("FORCE_COLOR") != "" || isCI() {
			Color = true
			color.NoColor = false // Force colors even without TTY
		}
		// Otherwise, let fatih/color auto-detect TTY
	} else {
		// Explicit config: sync with fatih/color
		color.NoColor = !Color
	}
}

// isCI returns true if running in a CI environment
func isCI() bool {
	ci, _ := strconv.ParseBool(os.Getenv("CI"))
	return ci
}

func Validate() error {
	if Download && Offline {
		return errors.New("task: You can't set both --download and --offline flags")
	}

	if Download && ClearCache {
		return errors.New("task: You can't set both --download and --clear-cache flags")
	}

	if Global && Dir != "" {
		return errors.New("task: You can't set both --global and --dir")
	}

	if Output.Name != "group" {
		if Output.Group.Begin != "" {
			return errors.New("task: You can't set --output-group-begin without --output=group")
		}
		if Output.Group.End != "" {
			return errors.New("task: You can't set --output-group-end without --output=group")
		}
		if Output.Group.ErrorOnly {
			return errors.New("task: You can't set --output-group-error-only without --output=group")
		}
	}

	if List && ListAll {
		return errors.New("task: cannot use --list and --list-all at the same time")
	}

	if ListJson && !List && !ListAll {
		return errors.New("task: --json only applies to --list or --list-all")
	}

	if NoStatus && !ListJson {
		return errors.New("task: --no-status only applies to --json with --list or --list-all")
	}

	if Nested && !ListJson {
		return errors.New("task: --nested only applies to --json with --list or --list-all")
	}

	return nil
}

// WithFlags is a special internal functional option that is used to pass flags
// from the CLI into any constructor that accepts functional options.
func WithFlags() task.ExecutorOption {
	return &flagsOption{}
}

type flagsOption struct{}

func (o *flagsOption) ApplyToExecutor(e *task.Executor) {
	// Set the sorter
	var sorter sort.Sorter
	switch TaskSort {
	case "none":
		sorter = sort.NoSort
	case "alphanumeric":
		sorter = sort.AlphaNumeric
	}

	// Change the directory to the user's home directory if the global flag is set
	dir := Dir
	if Global {
		home, err := os.UserHomeDir()
		if err == nil {
			dir = home
		}
	}

	e.Options(
		task.WithDir(dir),
		task.WithEntrypoint(Entrypoint),
		task.WithForce(Force),
		task.WithForceAll(ForceAll),
		task.WithInsecure(Insecure),
		task.WithDownload(Download),
		task.WithOffline(Offline),
		task.WithTrustedHosts(TrustedHosts),
		task.WithTimeout(Timeout),
		task.WithCacheExpiryDuration(CacheExpiryDuration),
		task.WithRemoteCacheDir(RemoteCacheDir),
		task.WithWatch(Watch),
		task.WithVerbose(Verbose),
		task.WithSilent(Silent),
		task.WithDisableFuzzy(DisableFuzzy),
		task.WithAssumeYes(AssumeYes),
		task.WithDry(Dry || Status),
		task.WithSummary(Summary),
		task.WithParallel(Parallel),
		task.WithColor(Color),
		task.WithConcurrency(Concurrency),
		task.WithInterval(Interval),
		task.WithOutputStyle(Output),
		task.WithTaskSorter(sorter),
		task.WithVersionCheck(true),
		task.WithFailfast(Failfast),
	)
}

// getConfig extracts a config value directly from a pointer field with a fallback default
func getConfig[T any](config *taskrcast.TaskRC, fieldFunc func() *T, fallback T) T {
	if config == nil {
		return fallback
	}

	field := fieldFunc()
	if field != nil {
		return *field
	}
	return fallback
}
