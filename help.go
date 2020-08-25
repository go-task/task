package task

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

// PrintTasksHelp prints help os tasks that have a description
func (e *Executor) PrintTasksHelp() {
	tasks := e.tasksWithDesc()
	if len(tasks) == 0 {
		e.Logger.Outf(logger.Yellow, "task: No tasks with description available")
		return
	}
	e.Logger.Outf(logger.Default, "task: Available tasks for this project:")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 0, '\t', 0)
	for _, task := range tasks {
		fmt.Fprintf(w, "* %s: \t%s\n", task.Name(), task.Desc)
	}
	w.Flush()
}

func (e *Executor) tasksWithDesc() (tasks []*taskfile.Task) {
	tasks = make([]*taskfile.Task, 0, len(e.Taskfile.Tasks))
	for _, task := range e.Taskfile.Tasks {
		if task != nil && task.Desc != "" {
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
