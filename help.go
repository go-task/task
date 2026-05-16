package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Ladicle/tabwriter"
	"golang.org/x/sync/errgroup"

	"github.com/go-task/task/v3/internal/editors"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/listing"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/taskfile/ast"
)

// ListOptions collects list-related options
type ListOptions struct {
	ListOnlyTasksWithDescriptions bool
	ListAllTasks                  bool
	FormatTaskListAsJSON          bool
	NoStatus                      bool
	Nested                        bool
	Long                          bool
	Tree                          bool
}

// NewListOptions creates a new ListOptions instance
func NewListOptions(list, listAll, listAsJson, noStatus, nested, long, tree bool) ListOptions {
	return ListOptions{
		ListOnlyTasksWithDescriptions: list,
		ListAllTasks:                  listAll,
		FormatTaskListAsJSON:          listAsJson,
		NoStatus:                      noStatus,
		Nested:                        nested,
		Long:                          long,
		Tree:                          tree,
	}
}

// ShouldListTasks returns true if one of the options to list tasks has been set to true
func (o ListOptions) ShouldListTasks() bool {
	return o.ListOnlyTasksWithDescriptions || o.ListAllTasks
}

// Filters returns the slice of FilterFunc which filters a list
// of ast.Task according to the given ListOptions
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
		output, err := e.ToEditorOutput(tasks, o.NoStatus, o.Nested, o.Long)
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
	if o.Tree {
		return e.listTasksTree(tasks, o)
	}
	e.Logger.Outf(logger.Default, "task: Available tasks for this project:\n")

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 6, ' ', 0)
	for _, task := range tasks {
		e.Logger.FOutf(w, logger.Yellow, "* ")
		e.Logger.FOutf(w, logger.Green, task.Task)
		desc := strings.ReplaceAll(task.Desc, "\n", " ")
		e.Logger.FOutf(w, logger.Default, ": \t%s", desc)
		if len(task.Aliases) > 0 {
			e.Logger.FOutf(w, logger.Cyan, "\t(aliases: %s)", strings.Join(task.Aliases, ", "))
		}
		_, _ = fmt.Fprint(w, "\n")
		e.writeTaskDetails(w, task, "  \t", o.Long)
	}
	if err := w.Flush(); err != nil {
		return false, err
	}
	return true, nil
}

func (e *Executor) writeTaskDetails(w io.Writer, task *ast.Task, indent string, long bool) {
	if listing.HasRequires(task) {
		e.Logger.FOutf(w, logger.Default, indent)
		e.Logger.FOutf(w, logger.Yellow, "requires:")
		e.Logger.FOutf(w, logger.Default, " %s\n", listing.FormatRequires(task.Requires))
	}
	if long {
		if deps := listing.FormatDeps(task.Deps); deps != "" {
			e.Logger.FOutf(w, logger.Default, indent)
			e.Logger.FOutf(w, logger.Yellow, "deps:")
			e.Logger.FOutf(w, logger.Default, " %s\n", deps)
		}
		if task.Summary != "" {
			summary := strings.TrimSpace(strings.ReplaceAll(task.Summary, "\n", " "))
			e.Logger.FOutf(w, logger.Default, indent)
			e.Logger.FOutf(w, logger.Yellow, "summary:")
			e.Logger.FOutf(w, logger.Default, " %s\n", summary)
		}
	}
}

// ListTaskNames prints only the task names in a Taskfile.
// Only tasks with a non-empty description are printed if allTasks is false.
// Otherwise, all task names are printed.
func (e *Executor) ListTaskNames(allTasks bool) error {
	// use stdout if no output defined
	var w io.Writer = os.Stdout
	if e.Stdout != nil {
		w = e.Stdout
	}

	// Sort the tasks
	if e.TaskSorter == nil {
		e.TaskSorter = sort.AlphaNumericWithRootTasksFirst
	}

	// Create a list of task names
	taskNames := make([]string, 0, e.Taskfile.Tasks.Len())
	for task := range e.Taskfile.Tasks.Values(e.TaskSorter) {
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
	return nil
}

func (e *Executor) listTasksTree(tasks []*ast.Task, o ListOptions) (bool, error) {
	e.Logger.Outf(logger.Default, "task: Available tasks for this project:\n")

	groups := listing.GroupByNamespace(tasks)
	hasNamespaced := listing.HasNamespacedGroups(groups)
	hasRoot := listing.HasRootGroup(groups)
	showSeparator := hasNamespaced && hasRoot

	// Move root group to end so namespaced groups appear first
	if showSeparator {
		for i, g := range groups {
			if g.Namespace == "" && i < len(groups)-1 {
				rootGroup := groups[i]
				groups = append(groups[:i], groups[i+1:]...)
				groups = append(groups, rootGroup)
				break
			}
		}
	}

	w := tabwriter.NewWriter(e.Stdout, 0, 8, 6, ' ', 0)
	firstGroup := true
	for _, group := range groups {
		isRoot := group.Namespace == ""
		if !firstGroup {
			_, _ = fmt.Fprint(w, "\n")
		}
		firstGroup = false
		if isRoot && showSeparator {
			e.Logger.FOutf(w, logger.Dim, "  ─────\n\n")
		}
		if !isRoot {
			e.Logger.FOutf(w, logger.Dim, "  %s\n", group.Namespace)
		}
		for _, task := range group.Tasks {
			name := group.LocalName(task)
			desc := strings.ReplaceAll(task.Desc, "\n", " ")
			indent := "  "
			if !isRoot {
				indent = "    "
			}
			nameColor := logger.Green
			if task.Internal {
				nameColor = logger.Dim
			}
			e.Logger.FOutf(w, nameColor, "%s%s", indent, name)
			e.Logger.FOutf(w, logger.Default, ":\t%s", desc)
			if len(task.Aliases) > 0 {
				e.Logger.FOutf(w, logger.Cyan, "\t(aliases: %s)", strings.Join(task.Aliases, ", "))
			}
			_, _ = fmt.Fprint(w, "\n")
			e.writeTaskDetails(w, task, indent+"  \t", o.Long)
		}
	}
	return true, w.Flush()
}

func (e *Executor) ToEditorOutput(tasks []*ast.Task, noStatus bool, nested bool, long bool) (*editors.Namespace, error) {
	var g errgroup.Group
	editorTasks := make([]editors.Task, len(tasks))

	// Look over each task in parallel and turn it into an editor task
	for i := range tasks {
		g.Go(func() error {
			var editorTask editors.Task
			if long {
				editorTask = editors.NewTaskLong(tasks[i])
			} else {
				editorTask = editors.NewTask(tasks[i])
			}

			if noStatus {
				editorTasks[i] = editorTask
				return nil
			}

			// Get the fingerprinting method to use
			method := e.Taskfile.Method
			if tasks[i].Method != "" {
				method = tasks[i].Method
			}
			upToDate, err := fingerprint.IsTaskUpToDate(context.Background(), tasks[i],
				fingerprint.WithMethod(method),
				fingerprint.WithTempDir(e.TempDir.Fingerprint),
				fingerprint.WithDry(e.Dry),
				fingerprint.WithLogger(e.Logger),
			)
			if err != nil {
				return err
			}

			editorTask.UpToDate = &upToDate
			editorTasks[i] = editorTask
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Create the root namespace
	var tasksLen int
	if !nested {
		tasksLen = len(editorTasks)
	}
	rootNamespace := &editors.Namespace{
		Tasks:    make([]editors.Task, tasksLen),
		Location: e.Taskfile.Location,
	}

	// Recursively add namespaces to the root namespace or if nesting is
	// disabled add them all to the root namespace
	for i, task := range editorTasks {
		taskNamespacePath := strings.Split(task.Task, ast.NamespaceSeparator)
		if nested {
			rootNamespace.AddNamespace(taskNamespacePath, task)
		} else {
			rootNamespace.Tasks[i] = task
		}
	}

	return rootNamespace, g.Wait()
}
