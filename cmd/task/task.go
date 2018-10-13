package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-task/task"
	"github.com/go-task/task/internal/args"

	"github.com/spf13/pflag"
)

var (
	version = "master"
)

const usage = `Usage: task [-ilfwvsd] [--init] [--list] [--force] [--watch] [--verbose] [--silent] [--dir] [--dry] [task...]

Runs the specified task(s). Falls back to the "default" task if no task name
was specified, or lists all tasks if an unknown task name was specified.

Example: 'task hello' with the following 'Taskfile.yml' file will generate an
'output.txt' file with the content "hello".

'''
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
		init        bool
		list        bool
		status      bool
		force       bool
		watch       bool
		verbose     bool
		silent      bool
		dry         bool
		dir         string
	)

	pflag.BoolVar(&versionFlag, "version", false, "show Task version")
	pflag.BoolVarP(&init, "init", "i", false, "creates a new Taskfile.yml in the current folder")
	pflag.BoolVarP(&list, "list", "l", false, "lists tasks with description of current Taskfile")
	pflag.BoolVar(&status, "status", false, "exits with non-zero exit code if any of the given tasks is not up-to-date")
	pflag.BoolVarP(&force, "force", "f", false, "forces execution even when the task is up-to-date")
	pflag.BoolVarP(&watch, "watch", "w", false, "enables watch of the given task")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enables verbose mode")
	pflag.BoolVarP(&silent, "silent", "s", false, "disables echoing")
	pflag.BoolVar(&dry, "dry", false, "compiles and prints tasks in the order that they would be run, without executing them")
	pflag.StringVarP(&dir, "dir", "d", "", "sets directory of execution")
	pflag.Parse()

	if versionFlag {
		log.Printf("Task version: %s\n", version)
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

	ctx := context.Background()
	if !watch {
		ctx = getSignalContext()
	}

	e := task.Executor{
		Force:   force,
		Watch:   watch,
		Verbose: verbose,
		Silent:  silent,
		Dir:     dir,
		Dry:     dry,

		Context: ctx,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := e.Setup(); err != nil {
		log.Fatal(err)
	}

	if list {
		e.PrintTasksHelp()
		return
	}

	arguments := pflag.Args()
	if len(arguments) == 0 {
		log.Println("task: No argument given, trying default task")
		arguments = []string{"default"}
	}

	calls, err := args.Parse(arguments...)
	if err != nil {
		log.Fatal(err)
	}

	if status {
		if err = e.Status(calls...); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := e.Run(calls...); err != nil {
		log.Fatal(err)
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
