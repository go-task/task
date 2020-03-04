package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	"github.com/go-task/task/v2"
	"github.com/go-task/task/v2/internal/args"

	"github.com/spf13/pflag"
)

var (
	version = "master"
	repo    = "go-task/task"
)

const usage = `Usage: task [-ilfwvsd] [--init] [--list] [--force] [--watch] [--verbose] [--silent] [--dir] [--dry] [--summary] [task...]

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

func doSelfUpdate(requireVersion string) {
	v, err := semver.Make(version)
	if err != nil {
		log.Println("Unable to detect version:", err)
		return
	}
	if requireVersion != "" {
		rv, err := semver.Make(requireVersion)
		if err != nil {
			log.Println("Invalid version requirement given:", err)
			return
		}
		if v.Equals(rv) {
			return
		}
	}
	latest, err := selfupdate.UpdateSelf(v, repo)
	if err != nil {
		log.Println("Binary update failed:", err)
		return
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Println("Current binary is the latest version", version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		log.Println("Release note:\n", latest.ReleaseNotes)
	}
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	pflag.Usage = func() {
		log.Print(usage)
		pflag.PrintDefaults()
	}

	var (
		versionFlag      bool
		init             bool
		list             bool
		listHidden       bool
		withDeps         bool
		status           bool
		force            bool
		watch            bool
		verbose          bool
		taskfileLocation string
		silent           bool
		dry              bool
		update           bool
		requireVersion   string
		dir              string
		summary          bool
		output           string
	)

	pflag.BoolVar(&versionFlag, "version", false, "show Task version")
	pflag.BoolVarP(&init, "init", "i", false, "creates a new Taskfile.yml in the current folder")
	pflag.BoolVarP(&list, "list", "l", false, "lists tasks of current Taskfile")
	pflag.BoolVar(&update, "update", false, "selfupdate task")
	pflag.StringVarP(&taskfileLocation, "config", "c", "Taskfile.yml", "Specify taskfile location")
	pflag.StringVar(&requireVersion, "require-version", "", "Expected version")
	pflag.BoolVar(&listHidden, "list-hidden",
		false, "lists all tasks")
	pflag.BoolVar(&withDeps, "with-deps", false, "list all tasks with dependencies")
	pflag.BoolVar(&status, "status", false, "exits with non-zero exit code if any of the given tasks is not up-to-date")
	pflag.BoolVarP(&force, "force", "f", false, "forces execution even when the task is up-to-date")
	pflag.BoolVarP(&watch, "watch", "w", false, "enables watch of the given task")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enables verbose mode")
	pflag.BoolVarP(&silent, "silent", "s", false, "disables echoing")
	pflag.BoolVar(&dry, "dry", false, "compiles and prints tasks in the order that they would be run, without executing them")
	pflag.BoolVar(&summary, "summary", false, "show summary about a task")
	pflag.StringVarP(&dir, "dir", "d", "", "sets directory of execution")
	pflag.StringVarP(&output, "output", "o", "", "sets output style: [interleaved|group|prefixed]")
	pflag.Parse()

	if versionFlag {
		log.Printf("Task version: %s\n", version)
		return
	}

	if update {
		doSelfUpdate(requireVersion)
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

	e := task.Executor{
		Force:            force,
		Watch:            watch,
		Verbose:          verbose,
		Silent:           silent,
		Dir:              dir,
		Dry:              dry,
		TaskfileLocation: taskfileLocation,
		Summary:          summary,
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,

		OutputStyle: output,
	}
	if err := e.Setup(); err != nil {
		log.Fatal(err)
	}

	if list {
		e.PrintTasksHelp(true, withDeps)
		return
	}

	if listHidden {
		e.PrintTasksHelp(false, withDeps)
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

	ctx := context.Background()
	if !watch {
		ctx = getSignalContext()
	}

	if status {
		if err = e.Status(ctx, calls...); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := e.Run(ctx, calls...); err != nil {
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
