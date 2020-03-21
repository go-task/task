package task

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/go-task/task/v2/internal/taskfile"
)

// PrintTasksHelp prints help os tasks that have a description
func (e *Executor) PrintTasksHelp() {
	tasks := e.tasksWithDesc()
	if len(tasks) == 0 {
		e.Logger.Outf("task: No tasks with description available")
		return
	}
	e.Logger.Outf("task: Available tasks for this project:")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 0, '\t', 0)
	for _, task := range tasks {
		fmt.Fprintf(w, "* %s: \t%s\n", task.Task, task.Desc)
	}
	w.Flush()
}

func (e *Executor) tasksWithDesc() (tasks []*taskfile.Task) {
	tasks = make([]*taskfile.Task, 0, len(e.Taskfile.Tasks))
	for _, task := range e.Taskfile.Tasks {
		if task.Desc != "" {
			compiledTask, err := e.CompiledTask(taskfile.Call{Task: task.Task})
			if err == nil {
				task = compiledTask
			}
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Task < tasks[j].Task })
	return
}

// PrintTaskNames prints only the task names in a taskfile.
func (e *Executor) PrintTaskNames() error {
	// if called from cmd/task.go, e.Taskfile has not yet been parsed
	if nil == e.Taskfile {
		err := e.readTaskfile()
		if nil != err {
			return err
		}
	}
	var w io.Writer = os.Stdout
	if nil != e.Stdout {
		w = e.Stdout
	}

	// create a slice from all map values
	task := make([]*taskfile.Task, 0, len(e.Taskfile.Tasks))
	for _, t := range e.Taskfile.Tasks {
		task = append(task, t)
	}

	sort.Slice(task,
		func(i, j int) bool {
			return task[i].Task < task[j].Task
		})
	for _, t := range task {
		fmt.Fprintf(w, "%s\n", strings.TrimSuffix(t.Task, ":"))
	}
	return nil
}
