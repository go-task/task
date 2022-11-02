package task

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/go-task/task/v3/internal/logger"
)

// ListTasks prints a list of tasks.
// Tasks that match the given filters will be excluded from the list.
// The function returns a boolean indicating whether or not tasks were found.
func (e *Executor) ListTasks(filters ...FilterFunc) bool {
	tasks := e.GetTaskList(filters...)
	if len(tasks) == 0 {
		return false
	}
	e.Logger.Outf(logger.Default, "task: Available tasks for this project:")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 6, ' ', 0)
	for _, task := range tasks {
		e.Logger.FOutf(w, logger.Yellow, "* ")
		e.Logger.FOutf(w, logger.Green, task.Task)
		e.Logger.FOutf(w, logger.Default, ": \t%s", task.Desc)
		if len(task.Aliases) > 0 {
			e.Logger.FOutf(w, logger.Cyan, "\t(aliases: %s)", strings.Join(task.Aliases, ", "))
		}
		fmt.Fprint(w, "\n")
	}
	w.Flush()
	return true
}

// ListTaskNames prints only the task names in a Taskfile.
// Only tasks with a non-empty description are printed if allTasks is false.
// Otherwise, all task names are printed.
func (e *Executor) ListTaskNames(allTasks bool) {
	// if called from cmd/task.go, e.Taskfile has not yet been parsed
	if e.Taskfile == nil {
		if err := e.readTaskfile(); err != nil {
			log.Fatal(err)
			return
		}
	}
	// use stdout if no output defined
	var w io.Writer = os.Stdout
	if e.Stdout != nil {
		w = e.Stdout
	}
	// create a string slice from all map values (*taskfile.Task)
	s := make([]string, 0, len(e.Taskfile.Tasks))
	for _, t := range e.Taskfile.Tasks {
		if (allTasks || t.Desc != "") && !t.Internal {
			s = append(s, strings.TrimRight(t.Task, ":"))
		}
	}
	// sort and print all task names
	sort.Strings(s)
	for _, t := range s {
		fmt.Fprintln(w, t)
	}
}
