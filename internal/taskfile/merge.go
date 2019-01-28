package taskfile

import (
	"fmt"
	"strings"
)

// NamespaceSeparator contains the character that separates namescapes
const NamespaceSeparator = ":"

// Merge merges the second Taskfile into the first
func Merge(t1, t2 *Taskfile, namespaces ...string) error {
	if t1.Version != t2.Version {
		return fmt.Errorf(`Taskfiles versions should match. First is "%s" but second is "%s"`, t1.Version, t2.Version)
	}

	if t2.Expansions != 0 && t2.Expansions != 2 {
		t1.Expansions = t2.Expansions
	}
	if t2.Output != "" {
		t1.Output = t2.Output
	}

	if t1.Includes == nil {
		t1.Includes = make(map[string]string)
	}
	for k, v := range t2.Includes {
		t1.Includes[k] = v
	}

	if t1.Vars == nil {
		t1.Vars = make(Vars)
	}
	for k, v := range t2.Vars {
		t1.Vars[k] = v
	}

	if t1.Env == nil {
		t1.Env = make(Vars)
	}
	for k, v := range t2.Env {
		t1.Env[k] = v
	}

	if t1.Tasks == nil {
		t1.Tasks = make(Tasks)
	}
	for k, v := range t2.Tasks {
		// FIXME(@andreynering): Refactor this block, otherwise we can
		// have serious side-effects in the future, since we're editing
		// the original references instead of deep copying them.

		t1.Tasks[taskNameWithNamespace(k, namespaces...)] = v

		for _, dep := range v.Deps {
			dep.Task = taskNameWithNamespace(dep.Task, namespaces...)
		}
		for _, cmd := range v.Cmds {
			if cmd.Task != "" {
				cmd.Task = taskNameWithNamespace(cmd.Task, namespaces...)
			}
		}
	}

	return nil
}

func taskNameWithNamespace(taskName string, namespaces ...string) string {
	return strings.Join(append(namespaces, taskName), NamespaceSeparator)
}
