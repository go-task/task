package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-task/task/v2"
	"github.com/go-task/task/v2/internal/args"
	"github.com/go-task/task/v2/internal/logger"

	"github.com/spf13/pflag"
)

var version = "master"

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
		versionFlag bool
		helpFlag    bool
		init        bool
		list        bool
		status      bool
		force       bool
		watch       bool
		verbose     bool
		silent      bool
		dry         bool
		summary     bool
		parallel    bool
		dir         string
		entrypoint  string
		output      string
		color       bool
	)

	pflag.BoolVar(&versionFlag, "version", false, "show Task version")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "shows Task usage")
	pflag.BoolVarP(&init, "init", "i", false, "creates a new Taskfile.yml in the current folder")
	pflag.BoolVarP(&list, "list", "l", false, "lists tasks with description of current Taskfile")
	pflag.BoolVar(&status, "status", false, "exits with non-zero exit code if any of the given tasks is not up-to-date")
	pflag.BoolVarP(&force, "force", "f", false, "forces execution even when the task is up-to-date")
	pflag.BoolVarP(&watch, "watch", "w", false, "enables watch of the given task")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enables verbose mode")
	pflag.BoolVarP(&silent, "silent", "s", false, "disables echoing")
	pflag.BoolVarP(&parallel, "parallel", "p", false, "executes tasks provided on command line in parallel")
	pflag.BoolVar(&dry, "dry", false, "compiles and prints tasks in the order that they would be run, without executing them")
	pflag.BoolVar(&summary, "summary", false, "show summary about a task")
	pflag.StringVarP(&dir, "dir", "d", "", "sets directory of execution")
	pflag.StringVarP(&entrypoint, "taskfile", "t", "", `choose which Taskfile to run. Defaults to "Taskfile.yml"`)
	pflag.StringVarP(&output, "output", "o", "", "sets output style: [interleaved|group|prefixed]")
	pflag.BoolVarP(&color, "color", "c", true, "colored output")
	pflag.Parse()

	if versionFlag {
		fmt.Printf("Task version: %s\n", version)
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

	if dir != "" && entrypoint != "" {
		log.Fatal("task: You can't set both --dir and --taskfile")
		return
	}
	if entrypoint != "" {
		dir = filepath.Dir(entrypoint)
		entrypoint = filepath.Base(entrypoint)
	} else {
		entrypoint = "Taskfile.yml"
	}

	e := task.Executor{
		Force:      force,
		Watch:      watch,
		Verbose:    verbose,
		Silent:     silent,
		Dir:        dir,
		Dry:        dry,
		Entrypoint: entrypoint,
		Summary:    summary,
		Parallel:   parallel,
		Color:      color,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,

		OutputStyle: output,
	}
	if err := e.Setup(); err != nil {
		log.Fatal(err)
	}

	if list {
		e.PrintTasksHelp()
		return
	}

	calls, globals := args.Parse(pflag.Args()...)
	e.Taskfile.Vars.Merge(globals)

	ctx := context.Background()
	if !watch {
		ctx = getSignalContext()
	}

	if status {
		if err := e.Status(ctx, calls...); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := e.Run(ctx, calls...); err != nil {
		e.Logger.Errf(logger.Red, "%v", err)
		os.Exit(1)
	}
}

func getSignalContext() context.Context {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-ch
		log.Printf("task: signal received: %s", sig)
		cancel()
	}()
	return ctx
}
