package task

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
)

var (
	// TaskFilePath is the default Taskfile
	TaskFilePath = "Taskfile.yml"
	// ShExists is true if Bash was found
	ShExists bool
	// ShPath constains the Bash path if found
	ShPath string

	// Tasks constains the tasks parsed from Taskfile
	Tasks = make(map[string]*Task)
)

func init() {
	var err error
	ShPath, err = exec.LookPath("sh")
	if err != nil {
		return
	}
	ShExists = true
}

// Task represents a task
type Task struct {
	Cmds      []string
	Deps      []string
	Sources   []string
	Generates []string
}

type taskNotFoundError struct {
	taskName string
}

func (err *taskNotFoundError) Error() string {
	return fmt.Sprintf(`Task "%s" not found`, err.taskName)
}

type taskRunError struct {
	taskName string
	err      error
}

func (err *taskRunError) Error() string {
	return fmt.Sprintf(`Failed to run task "%s": %v`, err.taskName, err.err)
}

// Run runs Task
func Run() {
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

	for _, a := range args {
		if err = RunTask(a); err != nil {
			log.Fatal(err)
		}
	}
}

// RunTask runs a task by its name
func RunTask(name string) error {
	t, ok := Tasks[name]
	if !ok {
		return &taskNotFoundError{name}
	}

	if isTaskUpToDate(t) {
		log.Printf(`Task "%s" is up to date`, name)
		return nil
	}

	for _, d := range t.Deps {
		if err := RunTask(d); err != nil {
			return err
		}
	}

	for _, c := range t.Cmds {
		if err := runCommand(c); err != nil {
			return &taskRunError{name, err}
		}
	}
	return nil
}

func isTaskUpToDate(t *Task) bool {
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

func runCommand(c string) error {
	var cmd *exec.Cmd
	if ShExists {
		cmd = exec.Command(ShPath, "-c", c)
	} else {
		cmd = exec.Command("cmd", "/C", c)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
