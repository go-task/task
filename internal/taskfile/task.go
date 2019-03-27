package taskfile

import (
	"strings"
)

// NamespaceSeparator contains the character that separates namescapes
const NamespaceSeparator = ":"

// Tasks representas a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task        string
	Cmds        []*Cmd
	Deps        []*Dep
	Desc        string
	Summary     string
	Sources     []string
	Generates   []string
	Status      []string
	Dir         string
	Vars        Vars
	Env         Vars
	Silent      bool
	Method      string
	Prefix      string
	Hidden      bool
	direct      bool
	IgnoreError bool `yaml:"ignore_error"`
}

// ApplyNamespace will update the task dependencies end returns new tasks
func (t *Task) ApplyNamespace(taskName string, ns ...string) []*Task {
	tasks := []*Task{}
	if len(ns) > 0 {
		if t.Hidden {
			ns[0] = strings.TrimPrefix(ns[0], ".")
		}

		if t.direct {
			ns[0] = strings.TrimPrefix(ns[0], "_")
			taskCopy := &Task{}
			*taskCopy = *t
			taskCopy.Task = taskName
			taskCopy.Hidden = false
			tasks = append(tasks, taskCopy)
		}
	}
	for _, cmd := range t.Cmds {
		if cmd.Task != "" {
			cmd.Task = taskWithNamespace(ns, cmd.Task)
		}
	}
	for _, dep := range t.Deps {
		if dep.Task != "" {
			dep.Task = taskWithNamespace(ns, dep.Task)
		}
	}
	t.Task = taskWithNamespace(ns, taskName)
	return append(tasks, t)
}

func taskWithNamespace(ns []string, taskName string) string {
	if strings.HasPrefix(taskName, ":") {
		return strings.TrimPrefix(taskName, ":")
	}
	return strings.Join(append(ns, taskName), NamespaceSeparator)
}
