package task

import (
	"fmt"
	"sort"
	"text/tabwriter"
)

func (e *Executor) printExistingTasksHelp() {
	tasks := e.tasksWithDesc()
	if len(tasks) == 0 {
		return
	}
	e.println("Available tasks for this project:")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 0, '\t', 0)
	for _, task := range tasks {
		fmt.Fprintln(w, fmt.Sprintf("- %s:\t%s", task, e.Tasks[task].Desc))
	}
	w.Flush()
}

func (e *Executor) tasksWithDesc() (tasks []string) {
	for name, task := range e.Tasks {
		if task.Desc != "" {
			tasks = append(tasks, name)
		}
	}
	sort.Strings(tasks)
	return
}
