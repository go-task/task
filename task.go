package main

import (
	"fmt"
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

	Tasks = make(map[string]*Task)
)

type Task struct {
	Cmds      []string
	Deps      []string
	Source    string
	Generates string
}

type TaskNotFoundError struct {
	taskName string
}

func (err *TaskNotFoundError) Error() string {
	return fmt.Sprintf(`Task "%s" not found`, err.taskName)
}

type TaskRunError struct {
	taskName string
	err      error
}

func (err *TaskRunError) Error() string {
	return fmt.Sprintf(`Failed to run task "%s": %v`, err.taskName, err.err)
}

func main() {
	log.SetFlags(0)

	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("No argument given")
	}

	file, err := ioutil.ReadFile(TaskFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal("Taskfile.yml not found")
		}
		log.Fatal(err)
	}

	if err = yaml.Unmarshal(file, &Tasks); err != nil {
		log.Fatal(err)
	}

	if err = RunTask(args[0]); err != nil {
		log.Fatal(err)
	}
}

func RunTask(name string) error {
	t, ok := Tasks[name]
	if !ok {
		return &TaskNotFoundError{name}
	}

	for _, d := range t.Deps {
		if err := RunTask(d); err != nil {
			return err
		}
	}

	for _, c := range t.Cmds {
		cmd := exec.Command("/bin/sh", "-c", c)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return &TaskRunError{name, err}
		}
	}
	return nil
}
