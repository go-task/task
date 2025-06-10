package task

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"strings"

	"github.com/Ladicle/tabwriter"
	"golang.org/x/sync/errgroup"

	sprig "github.com/go-task/slim-sprig/v3"

	"github.com/go-task/task/v3/internal/editors"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/sort"
	"github.com/go-task/task/v3/taskfile/ast"
)

var listDefaultTemplate = `task: Available tasks for this project:{{ range . }}
{{color "Yellow"}}* {{color "Green"}}{{ .Task }}{{color "Reset"}}:{{"\t"}} {{ .Desc | replace "\n" " " }}{{if len .Aliases}}{{color "Cyan"}}{{"\t"}}(aliases: {{.Aliases | join ", "}}){{color "Reset"}}{{end}}{{end}}
`

// ListOptions collects list-related options
type ListOptions struct {
	ListOnlyTasksWithDescriptions bool
	ListAllTasks                  bool
	FormatTaskListAsJSON          bool
	NoStatus                      bool
	ListTemplate                  string
}

// NewListOptions creates a new ListOptions instance
func NewListOptions(list, listAll, listAsJson, noStatus bool, template string) ListOptions {
	return ListOptions{
		ListOnlyTasksWithDescriptions: list,
		ListAllTasks:                  listAll,
		FormatTaskListAsJSON:          listAsJson,
		NoStatus:                      noStatus,
		ListTemplate:                  template,
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

	// List tasks using a template (optionally load from a file(s)).
	setupTemplate := func(listTemplate string, fallback string) (tmpl *template.Template, err error) {
		funcMap := template.FuncMap(sprig.TxtFuncMap())
		color := func(color string) string {
			if e.Logger.Color {
				switch color {
				case "Blue":
					return "\033[34m"
				case "Green":
					return "\033[32m"
				case "Cyan":
					return "\033[36m"
				case "Yellow":
					return "\033[33m"
				case "Magenta":
					return "\033[35m"
				case "Red":
					return "\033[31m"
				case "Reset":
					return "\033[0m"
				default:
					return "\033[0m"
				}
			} else {
				return ""
			}
		}
		funcMap["color"] = color
		if len(listTemplate) > 0 {
			files := strings.Split(listTemplate, ",")
			tmpl, err = template.New(path.Base(files[0])).Funcs(funcMap).ParseFiles(files...)
		} else {
			tmpl, err = template.New("list").Funcs(funcMap).Parse(fallback)
		}
		return
	}
	tmpl, err := setupTemplate(o.ListTemplate, listDefaultTemplate)
	if err != nil {
		return false, err
	}
	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(e.Stdout, 0, 8, 6, ' ', 0)
	if err := tmpl.Execute(w, tasks); err != nil {
		return false, err
	}
	if err := w.Flush(); err != nil {
		return false, err
	}
	return true, nil
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

func (e *Executor) ToEditorOutput(tasks []*ast.Task, noStatus bool) (*editors.Taskfile, error) {
	o := &editors.Taskfile{
		Tasks:    make([]editors.Task, len(tasks)),
		Location: e.Taskfile.Location,
	}
	var g errgroup.Group
	for i := range tasks {
		aliases := []string{}
		if len(tasks[i].Aliases) > 0 {
			aliases = tasks[i].Aliases
		}
		g.Go(func() error {
			o.Tasks[i] = editors.Task{
				Name:     tasks[i].Name(),
				Task:     tasks[i].Task,
				Desc:     tasks[i].Desc,
				Summary:  tasks[i].Summary,
				Aliases:  aliases,
				UpToDate: false,
				Location: &editors.Location{
					Line:     tasks[i].Location.Line,
					Column:   tasks[i].Location.Column,
					Taskfile: tasks[i].Location.Taskfile,
				},
			}

			if noStatus {
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

			o.Tasks[i].UpToDate = upToDate

			return nil
		})
	}
	return o, g.Wait()
}
