package task

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-task/task/execext"

	"github.com/spf13/pflag"
)

var (
	// TaskFilePath is the default Taskfile
	TaskFilePath = "Taskfile"

	// Force (--force or -f flag) forces a task to run even when it's up-to-date
	Force bool

	// Tasks constains the tasks parsed from Taskfile
	Tasks = make(map[string]*Task)

	runnedTasks = make(map[string]struct{})
)

// Task represents a task
type Task struct {
	Cmds      []string
	Deps      []string
	Sources   []string
	Generates []string
	Dir       string
	Vars      map[string]string
	Set       string
	Env       map[string]string
}

// Run runs Task
func Run() {
	log.SetFlags(0)

	args := pflag.Args()
	if len(args) == 0 {
		log.Println("task: No argument given, trying default task")
		args = []string{"default"}
	}

	var err error
	Tasks, err = readTaskfile()
	if err != nil {
		log.Fatal(err)
	}

	for _, a := range args {
		if err = RunTask(a); err != nil {
			log.Fatal(err)
		}
	}
}

// RunTask runs a task by its name
func RunTask(name string) error {
	if _, found := runnedTasks[name]; found {
		return &cyclicDepError{name}
	}
	runnedTasks[name] = struct{}{}

	t, ok := Tasks[name]
	if !ok {
		return &taskNotFoundError{name}
	}

	vars, err := t.handleVariables()
	if err != nil {
		return &taskRunError{name, err}
	}

	for _, d := range t.Deps {
		d, err = ReplaceVariables(d, vars)
		if err != nil {
			return err
		}
		if err = RunTask(d); err != nil {
			return err
		}
	}

	if !Force && t.isUpToDate() {
		log.Printf(`task: Task "%s" is up to date`, name)
		return nil
	}

	for i := range t.Cmds {
		if err = t.runCommand(i); err != nil {
			return &taskRunError{name, err}
		}
	}
	return nil
}

func (t *Task) isUpToDate() bool {
	if len(t.Sources) == 0 || len(t.Generates) == 0 {
		return false
	}

	sourcesMaxTime, err := getPatternsMaxTime(t.Sources)
	if err != nil || sourcesMaxTime.IsZero() {
		return false
	}

	generatesMinTime, err := getPatternsMinTime(t.Generates)
	if err != nil || generatesMinTime.IsZero() {
		return false
	}

	return generatesMinTime.After(sourcesMaxTime)
}

func (t *Task) runCommand(i int) error {
	vars, err := t.handleVariables()
	if err != nil {
		return err
	}
	c, err := ReplaceVariables(t.Cmds[i], vars)
	if err != nil {
		return err
	}
	dir, err := ReplaceVariables(t.Dir, vars)
	if err != nil {
		return err
	}
	cmd := execext.NewCommand(c)
	if dir != "" {
		cmd.Dir = dir
	}
	if t.Env != nil {
		cmd.Env = os.Environ()
		for key, value := range t.Env {
			replacedValue, err := ReplaceVariables(value, vars)
			if err != nil {
				return err
			}
			replacedKey, err := ReplaceVariables(key, vars)
			if err != nil {
				return err
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", replacedKey, replacedValue))
		}
	}
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if t.Set != "" {
		bytes, err := cmd.Output()
		if err != nil {
			return err
		}
		os.Setenv(t.Set, strings.TrimSpace(string(bytes)))
		return nil
	}
	cmd.Stdout = os.Stdout
	if err = cmd.Run(); err != nil {
		return err
	}
	return nil
}
