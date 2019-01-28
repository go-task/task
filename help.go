package task

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/go-task/task/v2/internal/taskfile"
)

// PrintTasksHelp prints help os tasks that have a description
func (e *Executor) PrintTasksHelp(skipHidden bool, withDeps bool) {
	tasks := e.tasksWithDesc()
	if len(tasks) == 0 {
		e.Logger.Outf("task: No tasks with description available")
		return
	}
	e.Logger.Outf("task: Available tasks for this project:")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 0, '\t', 0)
	for _, task := range tasks {
		if skipHidden && task.Hidden {
			continue
		}
		if task.Hidden {
			task.Desc = task.Desc + " [hidden]"
		}
		fmt.Fprintf(w, "* %s: \t%s\n", task.Task, task.Desc)
		if withDeps {
			for _, dep := range task.Deps {
				if dep.Task != "" {
					fmt.Fprintf(w, "  └─ %s\n", dep.Task)
				}
			}
			for _, cmd := range task.Cmds {
				if cmd.Task != "" {
					fmt.Fprintf(w, "  └─ %s\n", cmd.Task)
				}
			}
		}
	}
	w.Flush()
}

func (e *Executor) tasksWithDesc() (tasks []*taskfile.Task) {
	tasks = make([]*taskfile.Task, 0, len(e.Taskfile.Tasks))
	for _, task := range e.Taskfile.Tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Task < tasks[j].Task })
	return
}
