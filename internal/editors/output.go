package editors

import (
	"github.com/go-task/task/v3/taskfile/ast"
)

type (
	// Namespace wraps task list output for use in editor integrations (e.g. VSCode, etc)
	Namespace struct {
		Tasks      []Task                `json:"tasks"`
		Namespaces map[string]*Namespace `json:"namespaces,omitempty"`
		Location   string                `json:"location,omitempty"`
	}
	// Task describes a single task
	Task struct {
		Name     string    `json:"name"`
		Task     string    `json:"task"`
		Desc     string    `json:"desc"`
		Summary  string    `json:"summary"`
		Aliases  []string  `json:"aliases"`
		UpToDate *bool     `json:"up_to_date,omitempty"`
		Location *Location `json:"location"`
	}
	// Location describes a task's location in a taskfile
	Location struct {
		Line     int    `json:"line"`
		Column   int    `json:"column"`
		Taskfile string `json:"taskfile"`
	}
)

func NewTask(task *ast.Task) Task {
	aliases := []string{}
	if len(task.Aliases) > 0 {
		aliases = task.Aliases
	}
	return Task{
		Name:    task.Name(),
		Task:    task.Task,
		Desc:    task.Desc,
		Summary: task.Summary,
		Aliases: aliases,
		Location: &Location{
			Line:     task.Location.Line,
			Column:   task.Location.Column,
			Taskfile: task.Location.Taskfile,
		},
	}
}

func (parent *Namespace) AddNamespace(namespacePath []string, task Task) {
	if len(namespacePath) == 0 {
		return
	}

	// If there are no child namespaces, then we have found a task and we can
	// simply add it to the current namespace
	if len(namespacePath) == 1 {
		parent.Tasks = append(parent.Tasks, task)
		return
	}

	// Get the key of the current namespace in the path
	namespaceKey := namespacePath[0]

	// Add the namespace to the parent namespaces map using the namespace key
	if parent.Namespaces == nil {
		parent.Namespaces = make(map[string]*Namespace, 0)
	}

	// Search for the current namespace in the parent namespaces map
	// If it doesn't exist, create it
	namespace, ok := parent.Namespaces[namespaceKey]
	if !ok {
		namespace = &Namespace{}
		parent.Namespaces[namespaceKey] = namespace
	}

	// Remove the current namespace key from the namespace path.
	childNamespacePath := namespacePath[1:]

	// If there are no child namespaces in the task name, then we have found the
	// namespace of the task and we can add it to the current namespace.
	// Otherwise, we need to go deeper
	namespace.AddNamespace(childNamespacePath, task)
}
