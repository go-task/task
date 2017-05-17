package main

import (
	"fmt"

	"github.com/go-task/task"

	"github.com/spf13/pflag"
)

func main() {
	pflag.Usage = func() {
		fmt.Println(`task [target1 target2 ...]: Runs commands under targets like make.

Example: 'task hello' with the following 'Taskfile.yml' file will generate
an 'output.txt' file.
'''
hello:
  cmds:
    - echo "I am going to write a file named 'output.txt' now."
    - echo "hello" > output.txt
  generates:
    - output.txt
'''
`)
		pflag.PrintDefaults()
	}
	pflag.BoolVarP(&task.Init, "init", "i", false, "creates a new Taskfile.yml in the current folder")
	pflag.BoolVarP(&task.Force, "force", "f", false, "forces execution even when the task is up-to-date")
	pflag.BoolVarP(&task.Watch, "watch", "w", false, "enables watch of the given task")
	pflag.Parse()
	task.Run()
}
