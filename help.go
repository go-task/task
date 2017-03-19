package task

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
)

func printExistingTasksHelp() {
	tasks := tasksWithDesc()
	if len(tasks) == 0 {
		return
	}
	fmt.Println("Available tasks for this project:")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
	for _, task := range tasks {
		fmt.Fprintln(w, fmt.Sprintf("- %s:\t%s", task, Tasks[task].Desc))
	}
	w.Flush()
}

func tasksWithDesc() (tasks []string) {
	for name, task := range Tasks {
		if task.Desc != "" {
			tasks = append(tasks, name)
		}
	}
	sort.Strings(tasks)
	return
}
