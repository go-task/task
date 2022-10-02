package taskfile

import (
	"fmt"
	"strings"
)

// NamespaceSeparator contains the character that separates namescapes
const NamespaceSeparator = ":"

// Merge merges the second Taskfile into the first
func Merge(t1, t2 *Taskfile, includedTaskfile *IncludedTaskfile, namespaces ...string) error {
	if t1.Version != t2.Version {
		return fmt.Errorf(`task: Taskfiles versions should match. First is "%s" but second is "%s"`, t1.Version, t2.Version)
	}

	if t2.Expansions != 0 && t2.Expansions != 2 {
		t1.Expansions = t2.Expansions
	}
	if t2.Output.IsSet() {
		t1.Output = t2.Output
	}

	if t1.Includes == nil {
		t1.Includes = &IncludedTaskfiles{}
	}
	t1.Includes.Merge(t2.Includes)

	if t1.Vars == nil {
		t1.Vars = &Vars{}
	}
	if t1.Env == nil {
		t1.Env = &Vars{}
	}
	t1.Vars.Merge(t2.Vars)
	t1.Env.Merge(t2.Env)

	if t1.Tasks == nil {
		t1.Tasks = make(Tasks)
	}
	for k, v := range t2.Tasks {
		// FIXME(@andreynering): Refactor this block, otherwise we can
		// have serious side-effects in the future, since we're editing
		// the original references instead of deep copying them.
		task := v

		// Set the task to internal if EITHER the included task or the included
		// taskfile are marked as internal
		task.Internal = task.Internal || includedTaskfile.Internal

		// Deep copy the aliases so we can use them later
		origAliases := make([]string, len(task.Aliases))
		copy(origAliases, task.Aliases)

		// Add namespaces to dependencies, commands and aliases
		for _, dep := range task.Deps {
			dep.Task = taskNameWithNamespace(dep.Task, namespaces...)
		}
		for _, cmd := range task.Cmds {
			if cmd != nil && cmd.Task != "" {
				cmd.Task = taskNameWithNamespace(cmd.Task, namespaces...)
			}
		}
		for i, alias := range task.Aliases {
			task.Aliases[i] = taskNameWithNamespace(alias, namespaces...)
		}
		// Add namespace aliases
		if includedTaskfile != nil {
			for _, namespaceAlias := range includedTaskfile.Aliases {
				for _, alias := range origAliases {
					task.Aliases = append(v.Aliases, taskNameWithNamespace(alias, namespaceAlias))
				}
			}
		}

		// Add the task to the merged taskfile
		t1.Tasks[taskNameWithNamespace(k, namespaces...)] = task
	}

	return nil
}

func taskNameWithNamespace(taskName string, namespaces ...string) string {
	if strings.HasPrefix(taskName, ":") {
		return strings.TrimPrefix(taskName, ":")
	}
	return strings.Join(append(namespaces, taskName), NamespaceSeparator)
}
