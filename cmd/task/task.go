package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"mvdan.cc/sh/v3/syntax"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/internal/logger"
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
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	pflag.Usage = func() {
		log.Print(usage)
		pflag.PrintDefaults()
	}

	var (
		versionFlag   bool
		helpFlag      bool
		init          bool
		list          bool
		listAll       bool
		listJson      bool
		status        bool
		force         bool
		watch         bool
		verbose       bool
		silent        bool
		dry           bool
		summary       bool
		exitCode      bool
		parallel      bool
		concurrency   int
		concurrency64 int64
		dir           string
		entrypoint    string
		output        taskfile.Output
		color         bool
		interval      time.Duration
		global        bool
	)

	verbose, _ = strconv.ParseBool(getOsEnv("TASK_VERBOSE", "false"))
	force, _ = strconv.ParseBool(getOsEnv("TASK_FORCE", "false"))
	silent, _ = strconv.ParseBool(getOsEnv("TASK_SILENT", "false"))
	parallel, _ = strconv.ParseBool(getOsEnv("TASK_PARALLEL", "false"))
	dry, _ = strconv.ParseBool(getOsEnv("TASK_DRY_RUN", "false"))
	exitCode, _ = strconv.ParseBool(getOsEnv("TASK_EXIT_CODE", "false"))
	dir = getOsEnv("TASK_DIR", "")
	entrypoint = getOsEnv("TASK_TASKFILE", "")
	output.Name = getOsEnv("TASK_OUTPUT", "")
	output.Group.Begin = getOsEnv("TASK_OUTPUT_GROUP_BEGIN", "")
	output.Group.End = getOsEnv("TASK_OUTPUT_GROUP_END", "")
	output.Group.ErrorOnly, _ = strconv.ParseBool(getOsEnv("TASK_OUTPUT_GROUP_ERROR_ONLY", ""))
	concurrency64, _ = strconv.ParseInt(getOsEnv("TASK_CONCURRENCY", "0"), 10, 32)
	concurrency = int(concurrency64)
	global, _ = strconv.ParseBool(getOsEnv("TASK_GLOBAL", "false"))

	pflag.BoolVar(&versionFlag, "version", false, "Show Task version.")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "Shows Task usage.")
	pflag.BoolVarP(&init, "init", "i", false, "Creates a new Taskfile.yml in the current folder.")
	pflag.BoolVarP(&list, "list", "l", false, "Lists tasks with description of current Taskfile.")
	pflag.BoolVarP(&listAll, "list-all", "a", false, "Lists tasks with or without a description.")
	pflag.BoolVarP(&listJson, "json", "j", false, "Formats task list as JSON.")
	pflag.BoolVar(&status, "status", false, "Exits with non-zero exit code if any of the given tasks is not up-to-date.")
	pflag.BoolVarP(&force, "force", "f", force, "Forces execution even when the task is up-to-date.")
	pflag.BoolVarP(&watch, "watch", "w", false, "Enables watch of the given task.")
	pflag.BoolVarP(&verbose, "verbose", "v", verbose, "Enables verbose mode. (TASK_VERBOSE)")
	pflag.BoolVarP(&silent, "silent", "s", silent, "Disables echoing. (TASK_SILENT)")
	pflag.BoolVarP(&parallel, "parallel", "p", parallel, "Executes tasks provided on command line in parallel. (TASK_PARALLEL)")
	pflag.BoolVarP(&dry, "dry", "n", dry, "Compiles and prints tasks in the order that they would be run, without executing them. (TASK_DRY)")
	pflag.BoolVar(&summary, "summary", false, "Show summary about a task.")
	pflag.BoolVarP(&exitCode, "exit-code", "x", exitCode, "Pass-through the exit code of the task command. (TASK_EXIT_CODE)")
	pflag.StringVarP(&dir, "dir", "d", dir, "Sets directory of execution. (TASK_DIR)")
	pflag.StringVarP(&entrypoint, "taskfile", "t", entrypoint, `Choose which Taskfile to run. Defaults to "Taskfile.yml". (TASK_TASKFILE)`)
	pflag.StringVarP(&output.Name, "output", "o", output.Name, "Sets output style: [interleaved|group|prefixed]. (TASK_OUTPUT)")
	pflag.StringVar(&output.Group.Begin, "output-group-begin", output.Group.Begin, "Message template to print before a task's grouped output. (TASK_OUTPUT_GROUP_BEGIN)")
	pflag.StringVar(&output.Group.End, "output-group-end", output.Group.End, "Message template to print after a task's grouped output. (TASK_OUTPUT_GROUP_END)")
	pflag.BoolVar(&output.Group.ErrorOnly, "output-group-error-only", output.Group.ErrorOnly, "Swallow output from successful tasks. (TASK_OUTPUT_GROUP_ERROR_ONLY)")
	pflag.BoolVarP(&color, "color", "c", true, "Colored output. Enabled by default. Set flag to false or use NO_COLOR=1 to disable. (NO_COLOR)")
	pflag.IntVarP(&concurrency, "concurrency", "C", concurrency, "Limit number tasks to run concurrently. (TASK_CONCURRENCY)")
	pflag.DurationVarP(&interval, "interval", "I", 0, "Interval to watch for changes.")
	pflag.BoolVarP(&global, "global", "g", global, "Runs global Taskfile, from $HOME/Taskfile.{yml,yaml}. (TASK_GLOBAL)")
	pflag.Parse()

	if versionFlag {
		fmt.Printf("Task version: %s\n", ver.GetVersion())
		return
	}

	if helpFlag {
		pflag.Usage()
		return
	}

	if init {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		if err := task.InitTaskfile(os.Stdout, wd); err != nil {
			log.Fatal(err)
		}
		return
	}

	if global && dir != "" {
		log.Fatal("task: You can't set both --global and --dir")
		return
	}
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("task: Failed to get user home directory: %w", err)
			return
		}
		dir = home
	}

	if dir != "" && entrypoint != "" {
		log.Fatal("task: You can't set both --dir and --taskfile")
		return
	}
	if entrypoint != "" {
		dir = filepath.Dir(entrypoint)
		entrypoint = filepath.Base(entrypoint)
	}

	if output.Name != "group" {
		if output.Group.Begin != "" {
			log.Fatal("task: You can't set --output-group-begin without --output=group")
			return
		}
		if output.Group.End != "" {
			log.Fatal("task: You can't set --output-group-end without --output=group")
			return
		}
		if output.Group.ErrorOnly {
			log.Fatal("task: You can't set --output-group-error-only without --output=group")
			return
		}
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
	}

	var listOptions = task.NewListOptions(list, listAll, listJson)
	if err := listOptions.Validate(); err != nil {
		log.Fatal(err)
	}

	if (listOptions.ShouldListTasks()) && silent {
		e.ListTaskNames(listAll)
		return
	}

	if err := e.Setup(); err != nil {
		log.Fatal(err)
	}

	if listOptions.ShouldListTasks() {
		if foundTasks, err := e.ListTasks(listOptions); !foundTasks || err != nil {
			os.Exit(1)
		}
		return
	}

	var (
		calls   []taskfile.Call
		globals *taskfile.Vars
	)

	tasksAndVars, cliArgs, err := getArgs()
	if err != nil {
		log.Fatal(err)
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
		if err := e.Status(ctx, calls...); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := e.Run(ctx, calls...); err != nil {
		e.Logger.Errf(logger.Red, "%v", err)

		if exitCode {
			if err, ok := err.(*task.TaskRunError); ok {
				os.Exit(err.ExitCode())
			}
		}
		os.Exit(1)
	}
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

func getOsEnv(key string, defaultValue string) string {
	if res, err := os.LookupEnv(key); err {
		return res
	}
	return defaultValue
}
