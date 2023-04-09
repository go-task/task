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

func main() {
	if err := run(); err != nil {
		if err, ok := err.(errors.TaskError); ok {
			log.Print(err.Error())
			os.Exit(err.Code())
		}
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

	var (
		versionFlag bool
		helpFlag    bool
		init        bool
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
	)

	pflag.BoolVar(&versionFlag, "version", false, "Show Task version.")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "Shows Task usage.")
	pflag.BoolVarP(&init, "init", "i", false, "Creates a new Taskfile.yml in the current folder.")
	pflag.BoolVarP(&list, "list", "l", false, "Lists tasks with description of current Taskfile.")
	pflag.BoolVarP(&listAll, "list-all", "a", false, "Lists tasks with or without a description.")
	pflag.BoolVarP(&listJson, "json", "j", false, "Formats task list as JSON.")
	pflag.StringVar(&taskSort, "sort", "", "Changes the order of the tasks when listed.")
	pflag.BoolVar(&status, "status", false, "Exits with non-zero exit code if any of the given tasks is not up-to-date.")
	pflag.BoolVarP(&force, "force", "f", false, "Forces execution even when the task is up-to-date.")
	pflag.BoolVarP(&watch, "watch", "w", false, "Enables watch of the given task.")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "Enables verbose mode.")
	pflag.BoolVarP(&silent, "silent", "s", false, "Disables echoing.")
	pflag.BoolVarP(&parallel, "parallel", "p", false, "Executes tasks provided on command line in parallel.")
	pflag.BoolVarP(&dry, "dry", "n", false, "Compiles and prints tasks in the order that they would be run, without executing them.")
	pflag.BoolVar(&summary, "summary", false, "Show summary about a task.")
	pflag.BoolVarP(&exitCode, "exit-code", "x", false, "Pass-through the exit code of the task command.")
	pflag.StringVarP(&dir, "dir", "d", "", "Sets directory of execution.")
	pflag.StringVarP(&entrypoint, "taskfile", "t", "", `Choose which Taskfile to run. Defaults to "Taskfile.yml".`)
	pflag.StringVarP(&output.Name, "output", "o", "", "Sets output style: [interleaved|group|prefixed].")
	pflag.StringVar(&output.Group.Begin, "output-group-begin", "", "Message template to print before a task's grouped output.")
	pflag.StringVar(&output.Group.End, "output-group-end", "", "Message template to print after a task's grouped output.")
	pflag.BoolVar(&output.Group.ErrorOnly, "output-group-error-only", false, "Swallow output from successful tasks.")
	pflag.BoolVarP(&color, "color", "c", true, "Colored output. Enabled by default. Set flag to false or use NO_COLOR=1 to disable.")
	pflag.IntVarP(&concurrency, "concurrency", "C", 0, "Limit number tasks to run concurrently.")
	pflag.DurationVarP(&interval, "interval", "I", 0, "Interval to watch for changes.")
	pflag.BoolVarP(&global, "global", "g", false, "Runs global Taskfile, from $HOME/Taskfile.{yml,yaml}.")
	pflag.Parse()

	if versionFlag {
		fmt.Printf("Task version: %s\n", ver.GetVersion())
		return nil
	}

	if helpFlag {
		pflag.Usage()
		return nil
	}

	if init {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		if err := task.InitTaskfile(os.Stdout, wd); err != nil {
			log.Fatal(err)
		}
		return nil
	}

	if global && dir != "" {
		log.Fatal("task: You can't set both --global and --dir")
		return nil
	}
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("task: Failed to get user home directory: %w", err)
		}
		dir = home
	}

	if dir != "" && entrypoint != "" {
		return errors.New("task: You can't set both --dir and --taskfile")
	}
	if entrypoint != "" {
		dir = filepath.Dir(entrypoint)
		entrypoint = filepath.Base(entrypoint)
	}

	if output.Name != "group" {
		if output.Group.Begin != "" {
			return errors.New("task: You can't set --output-group-begin without --output=group")
		}
		if output.Group.End != "" {
			return errors.New("task: You can't set --output-group-end without --output=group")
		}
		if output.Group.ErrorOnly {
			return errors.New("task: You can't set --output-group-error-only without --output=group")
		}
	}

	var taskSorter sort.TaskSorter
	switch taskSort {
	case "none":
		taskSorter = &sort.Noop{}
	case "alphanumeric":
		taskSorter = &sort.AlphaNumeric{}
	}

	e := task.Executor{
		Force:       force,
		Watch:       watch,
		Verbose:     verbose,
		Silent:      silent,
		Dir:         dir,
		Dry:         dry,
		Entrypoint:  entrypoint,
		Summary:     summary,
		Parallel:    parallel,
		Color:       color,
		Concurrency: concurrency,
		Interval:    interval,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,

		OutputStyle: output,
		TaskSorter:  taskSorter,
	}

	listOptions := task.NewListOptions(list, listAll, listJson)
	if err := listOptions.Validate(); err != nil {
		return err
	}

	if (listOptions.ShouldListTasks()) && silent {
		e.ListTaskNames(listAll)
		return nil
	}

	if err := e.Setup(); err != nil {
		return err
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

	if !watch {
		e.InterceptInterruptSignals()
	}

	ctx := context.Background()

	if status {
		return e.Status(ctx, calls...)
	}

	if err := e.Run(ctx, calls...); err != nil {
		e.Logger.Errf(logger.Red, "%v", err)

		if exitCode {
			if err, ok := err.(*errors.TaskRunError); ok {
				os.Exit(err.TaskExitCode())
			}
		}
		return err
	}
	return nil
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
