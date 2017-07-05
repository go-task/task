package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-task/task"

	"github.com/spf13/pflag"
)

var (
	version = "master"
)

const usage = `Usage: task [-ifwv] [--init] [--force] [--watch] [--verbose] [task...]

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
`

func main() {
	log.SetFlags(0)

	pflag.Usage = func() {
		fmt.Println(usage)
		pflag.PrintDefaults()
	}

	var (
		versionFlag bool
		init        bool
		force       bool
		watch       bool
		verbose     bool
	)

	pflag.BoolVar(&versionFlag, "version", false, "show Task version")
	pflag.BoolVarP(&init, "init", "i", false, "creates a new Taskfile.yml in the current folder")
	pflag.BoolVarP(&force, "force", "f", false, "forces execution even when the task is up-to-date")
	pflag.BoolVarP(&watch, "watch", "w", false, "enables watch of the given task")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enables verbose mode")
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
		if err := task.InitTaskfile(wd); err != nil {
			log.Fatal(err)
		}
		return
	}

	e := task.Executor{
		Force:   force,
		Watch:   watch,
		Verbose: verbose,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := e.ReadTaskfile(); err != nil {
		log.Fatal(err)
	}

	args := pflag.Args()
	if len(args) == 0 {
		log.Println("task: No argument given, trying default task")
		args = []string{"default"}
	}

	if err := e.Run(args...); err != nil {
		log.Fatal(err)
	}
}
