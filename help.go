package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/sync/errgroup"

	"github.com/go-task/task/v3/internal/editors"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/taskfile"
)

// ListOptions collects list-related options
type ListOptions struct {
	ListOnlyTasksWithDescriptions bool
	ListAllTasks                  bool
	FormatTaskListAsJSON          bool
	NoStatus                      bool
}

// NewListOptions creates a new ListOptions instance
func NewListOptions(list, listAll, listAsJson, noStatus bool) ListOptions {
	return ListOptions{
		ListOnlyTasksWithDescriptions: list,
		ListAllTasks:                  listAll,
		FormatTaskListAsJSON:          listAsJson,
		NoStatus:                      noStatus,
	}
}

// ShouldListTasks returns true if one of the options to list tasks has been set to true
func (o ListOptions) ShouldListTasks() bool {
	return o.ListOnlyTasksWithDescriptions || o.ListAllTasks
}

// Validate validates that the collection of list-related options are in a valid configuration
func (o ListOptions) Validate() error {
	if o.ListOnlyTasksWithDescriptions && o.ListAllTasks {
		return fmt.Errorf("task: cannot use --list and --list-all at the same time")
	}
	if o.FormatTaskListAsJSON && !o.ShouldListTasks() {
		return fmt.Errorf("task: --json only applies to --list or --list-all")
	}
	if o.NoStatus && !o.FormatTaskListAsJSON {
		return fmt.Errorf("task: --no-status only applies to --json with --list or --list-all")
	}
	return nil
}

// Filters returns the slice of FilterFunc which filters a list
// of taskfile.Task according to the given ListOptions
func (o ListOptions) Filters() []FilterFunc {
	filters := []FilterFunc{FilterOutInternal}

	if o.ListOnlyTasksWithDescriptions {
		filters = append(filters, FilterOutNoDesc)
	}

	return filters
}

// ListTasks prints a list of tasks.
// Tasks that match the given filters will be excluded from the list.
// The function returns a boolean indicating whether tasks were found
// and an error if one was encountered while preparing the output.
func (e *Executor) ListTasks(o ListOptions) (bool, error) {
	tasks, err := e.GetTaskList(o.Filters()...)
	if err != nil {
		return false, err
	}
	if o.FormatTaskListAsJSON {
		output, err := e.ToEditorOutput(tasks, o.NoStatus)
		if err != nil {
			return false, err
		}

		encoder := json.NewEncoder(e.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return false, err
		}

		return len(tasks) > 0, nil
	}
	if len(tasks) == 0 {
		if o.ListOnlyTasksWithDescriptions {
			e.Logger.Outf(logger.Yellow, "task: No tasks with description available. Try --list-all to list all tasks\n")
		} else if o.ListAllTasks {
			e.Logger.Outf(logger.Yellow, "task: No tasks available\n")
		}
		return false, nil
	}
	e.Logger.Outf(logger.Default, "task: Available tasks for this project:\n")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 6, ' ', 0)
	for _, task := range tasks {
		e.Logger.FOutf(w, logger.Yellow, "* ")
		e.Logger.FOutf(w, logger.Green, task.Task)
		e.Logger.FOutf(w, logger.Default, ": \t%s", task.Desc)
		if len(task.Aliases) > 0 {
			e.Logger.FOutf(w, logger.Cyan, "\t(aliases: %s)", strings.Join(task.Aliases, ", "))
		}
		_, _ = fmt.Fprint(w, "\n")
	}
	if err := w.Flush(); err != nil {
		return false, err
	}
	return true, nil
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

	// Get the list of tasks and sort them
	tasks := e.Taskfile.Tasks.Values()

	// Sort the tasks
	if e.TaskSorter == nil {
		e.TaskSorter = &sort.AlphaNumericWithRootTasksFirst{}
	}
	e.TaskSorter.Sort(tasks)

	// Create a list of task names
	taskNames := make([]string, 0, e.Taskfile.Tasks.Len())
	for _, task := range tasks {
		if (allTasks || task.Desc != "") && !task.Internal {
			taskNames = append(taskNames, strings.TrimRight(task.Task, ":"))
			for _, alias := range task.Aliases {
				taskNames = append(taskNames, strings.TrimRight(alias, ":"))
			}
		}
	}
	for _, t := range taskNames {
		fmt.Fprintln(w, t)
	}
}

func (e *Executor) ToEditorOutput(tasks []*taskfile.Task, noStatus bool) (*editors.Taskfile, error) {
	o := &editors.Taskfile{
		Tasks:    make([]editors.Task, len(tasks)),
		Location: e.Taskfile.Location,
	}
	var g errgroup.Group
	for i := range tasks {
		task := tasks[i]
		j := i
		g.Go(func() error {
			o.Tasks[j] = editors.Task{
				Name:     task.Name(),
				Desc:     task.Desc,
				Summary:  task.Summary,
				UpToDate: false,
				Location: &editors.Location{
					Line:     task.Location.Line,
					Column:   task.Location.Column,
					Taskfile: task.Location.Taskfile,
				},
			}

			if noStatus {
				return nil
			}

			// Get the fingerprinting method to use
			method := e.Taskfile.Method
			if task.Method != "" {
				method = task.Method
			}
			upToDate, err := fingerprint.IsTaskUpToDate(context.Background(), task,
				fingerprint.WithMethod(method),
				fingerprint.WithTempDir(e.TempDir),
				fingerprint.WithDry(e.Dry),
				fingerprint.WithLogger(e.Logger),
			)
			if err != nil {
				return err
			}

			o.Tasks[j].UpToDate = upToDate

			return nil
		})
	}
	return o, g.Wait()
}
