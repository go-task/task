package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kardianos/osext"
	"gopkg.in/yaml.v2"
)

var (
	CurrentDirectory, _ = osext.ExecutableFolder()
	TaskFilePath        = filepath.Join(CurrentDirectory, "Taskfile.yml")
)

type Task struct {
	Cmds      []string
	Deps      []string
	Source    string
	Generates string
}

func main() {
	log.SetFlags(0)

	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("No argument given")
	}

	file, err := ioutil.ReadFile(TaskFilePath)
	if err != nil {
		log.Fatal(err)
	}

	tasks := make(map[string]*Task)
	if err = yaml.Unmarshal(file, &tasks); err != nil {
		log.Fatal(err)
	}

	task, ok := tasks[args[0]]
	if !ok {
		log.Fatalf(`Task "%s" not found`, args[0])
	}

	if err = RunTask(task); err != nil {
		log.Fatal(err)
	}
}

func RunTask(t *Task) error {
	for _, c := range t.Cmds {
		cmd := exec.Command("/bin/sh", "-c", c)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
