package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"mvdan.cc/sh/v3/syntax"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/completion"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/sort"
	ver "github.com/go-task/task/v3/internal/version"
	"github.com/go-task/task/v3/taskfile"
)

const usage = `Usage: task [-ilfwvsd] [--init] [--list] [--force] [--watch] [--verbose] [--silent] [--dir] [--taskfile] [--dry] [--summary] [task...]

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

var flags struct {
	version     bool
	help        bool
	init        bool
	completion  string
	list        bool
	listAll     bool
	listJson    bool
	taskSort    string
	status      bool
	force       bool
	watch       bool
	verbose     bool
	silent      bool
	dry         bool
	summary     bool
	exitCode    bool
	parallel    bool
	concurrency int
	dir         string
	entrypoint  string
	output      taskfile.Output
	color       bool
	interval    time.Duration
	global      bool
}

func main() {
	if err := run(); err != nil {
		l := &logger.Logger{
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Verbose: flags.verbose,
			Color:   flags.color,
		}

		if err, ok := err.(*errors.TaskRunError); ok && flags.exitCode {
			l.Errf(logger.Red, "%v\n", err)
			os.Exit(err.TaskExitCode())
		}
		if err, ok := err.(errors.TaskError); ok {
			l.Errf(logger.Red, "%v\n", err)
			os.Exit(err.Code())
		}
		l.Errf(logger.Red, "%v\n", err)
		os.Exit(errors.CodeUnknown)
	}
	os.Exit(errors.CodeOk)
}

func run() error {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	pflag.Usage = func() {
		log.Print(usage)
		pflag.PrintDefaults()
	}

	pflag.BoolVar(&flags.version, "version", false, "Show Task version.")
	pflag.BoolVarP(&flags.help, "help", "h", false, "Shows Task usage.")
	pflag.BoolVarP(&flags.init, "init", "i", false, "Creates a new Taskfile.yml in the current folder.")
	pflag.StringVar(&flags.completion, "completion", "", "Generates shell completion script.")
	pflag.BoolVarP(&flags.list, "list", "l", false, "Lists tasks with description of current Taskfile.")
	pflag.BoolVarP(&flags.listAll, "list-all", "a", false, "Lists tasks with or without a description.")
	pflag.BoolVarP(&flags.listJson, "json", "j", false, "Formats task list as JSON.")
	pflag.StringVar(&flags.taskSort, "sort", "", "Changes the order of the tasks when listed.")
	pflag.BoolVar(&flags.status, "status", false, "Exits with non-zero exit code if any of the given tasks is not up-to-date.")
	pflag.BoolVarP(&flags.force, "force", "f", false, "Forces execution even when the task is up-to-date.")
	pflag.BoolVarP(&flags.watch, "watch", "w", false, "Enables watch of the given task.")
	pflag.BoolVarP(&flags.verbose, "verbose", "v", false, "Enables verbose mode.")
	pflag.BoolVarP(&flags.silent, "silent", "s", false, "Disables echoing.")
	pflag.BoolVarP(&flags.parallel, "parallel", "p", false, "Executes tasks provided on command line in parallel.")
	pflag.BoolVarP(&flags.dry, "dry", "n", false, "Compiles and prints tasks in the order that they would be run, without executing them.")
	pflag.BoolVar(&flags.summary, "summary", false, "Show summary about a task.")
	pflag.BoolVarP(&flags.exitCode, "exit-code", "x", false, "Pass-through the exit code of the task command.")
	pflag.StringVarP(&flags.dir, "dir", "d", "", "Sets directory of execution.")
	pflag.StringVarP(&flags.entrypoint, "taskfile", "t", "", `Choose which Taskfile to run. Defaults to "Taskfile.yml".`)
	pflag.StringVarP(&flags.output.Name, "output", "o", "", "Sets output style: [interleaved|group|prefixed].")
	pflag.StringVar(&flags.output.Group.Begin, "output-group-begin", "", "Message template to print before a task's grouped output.")
	pflag.StringVar(&flags.output.Group.End, "output-group-end", "", "Message template to print after a task's grouped output.")
	pflag.BoolVar(&flags.output.Group.ErrorOnly, "output-group-error-only", false, "Swallow output from successful tasks.")
	pflag.BoolVarP(&flags.color, "color", "c", true, "Colored output. Enabled by default. Set flag to false or use NO_COLOR=1 to disable.")
	pflag.IntVarP(&flags.concurrency, "concurrency", "C", 0, "Limit number tasks to run concurrently.")
	pflag.DurationVarP(&flags.interval, "interval", "I", 0, "Interval to watch for changes.")
	pflag.BoolVarP(&flags.global, "global", "g", false, "Runs global Taskfile, from $HOME/Taskfile.{yml,yaml}.")
	pflag.Parse()

	if flags.version {
		fmt.Printf("Task version: %s\n", ver.GetVersion())
		return nil
	}

	if flags.help {
		pflag.Usage()
		return nil
	}

	if flags.init {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		if err := task.InitTaskfile(os.Stdout, wd); err != nil {
			log.Fatal(err)
		}
		return nil
	}

	if flags.global && flags.dir != "" {
		log.Fatal("task: You can't set both --global and --dir")
		return nil
	}
	if flags.global {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("task: Failed to get user home directory: %w", err)
		}
		flags.dir = home
	}

	if flags.dir != "" && flags.entrypoint != "" {
		return errors.New("task: You can't set both --dir and --taskfile")
	}
	if flags.entrypoint != "" {
		flags.dir = filepath.Dir(flags.entrypoint)
		flags.entrypoint = filepath.Base(flags.entrypoint)
	}

	if flags.output.Name != "group" {
		if flags.output.Group.Begin != "" {
			return errors.New("task: You can't set --output-group-begin without --output=group")
		}
		if flags.output.Group.End != "" {
			return errors.New("task: You can't set --output-group-end without --output=group")
		}
		if flags.output.Group.ErrorOnly {
			return errors.New("task: You can't set --output-group-error-only without --output=group")
		}
	}

	var taskSorter sort.TaskSorter
	switch flags.taskSort {
	case "none":
		taskSorter = &sort.Noop{}
	case "alphanumeric":
		taskSorter = &sort.AlphaNumeric{}
	}

	e := task.Executor{
		Force:       flags.force,
		Watch:       flags.watch,
		Verbose:     flags.verbose,
		Silent:      flags.silent,
		Dir:         flags.dir,
		Dry:         flags.dry,
		Entrypoint:  flags.entrypoint,
		Summary:     flags.summary,
		Parallel:    flags.parallel,
		Color:       flags.color,
		Concurrency: flags.concurrency,
		Interval:    flags.interval,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,

		OutputStyle: flags.output,
		TaskSorter:  taskSorter,
	}

	listOptions := task.NewListOptions(flags.list, flags.listAll, flags.listJson)
	if err := listOptions.Validate(); err != nil {
		return err
	}

	if (listOptions.ShouldListTasks()) && flags.silent {
		e.ListTaskNames(flags.listAll)
		return nil
	}

	if err := e.Setup(); err != nil {
		return err
	}

	if flags.completion != "" {
		script, err := completion.Compile(flags.completion, e.Taskfile.Tasks)
		if err != nil {
			return err
		}
		fmt.Println(script)
		return nil
	}

	if listOptions.ShouldListTasks() {
		foundTasks, err := e.ListTasks(listOptions)
		if err != nil {
			return err
		}
		if !foundTasks {
			os.Exit(errors.CodeUnknown)
		}
		return nil
	}

	var (
		calls   []taskfile.Call
		globals *taskfile.Vars
	)

	tasksAndVars, cliArgs, err := getArgs()
	if err != nil {
		return err
	}

	if e.Taskfile.Version.Compare(taskfile.V3) >= 0 {
		calls, globals = args.ParseV3(tasksAndVars...)
	} else {
		calls, globals = args.ParseV2(tasksAndVars...)
	}

	globals.Set("CLI_ARGS", taskfile.Var{Static: cliArgs})
	e.Taskfile.Vars.Merge(globals)

	if !flags.watch {
		e.InterceptInterruptSignals()
	}

	ctx := context.Background()

	if flags.status {
		return e.Status(ctx, calls...)
	}

	return e.Run(ctx, calls...)
}

func getArgs() ([]string, string, error) {
	var (
		args          = pflag.Args()
		doubleDashPos = pflag.CommandLine.ArgsLenAtDash()
	)

	if doubleDashPos == -1 {
		return args, "", nil
	}

	var quotedCliArgs []string
	for _, arg := range args[doubleDashPos:] {
		quotedCliArg, err := syntax.Quote(arg, syntax.LangBash)
		if err != nil {
			return nil, "", err
		}
		quotedCliArgs = append(quotedCliArgs, quotedCliArg)
	}
	return args[:doubleDashPos], strings.Join(quotedCliArgs, " "), nil
}
