package taskfile

import (
	"fmt"
	"strings"
)

// NamespaceSeparator contains the character that separates namespaces
const NamespaceSeparator = ":"

// Merge merges the second Taskfile into the first
func Merge(t1, t2 *Taskfile, includedTaskfile *IncludedTaskfile, namespaces ...string) error {
	if !t1.Version.Equal(t2.Version) {
		return fmt.Errorf(`task: Taskfiles versions should match. First is "%s" but second is "%s"`, t1.Version, t2.Version)
	}

	if t2.Expansions != 0 && t2.Expansions != 2 {
		t1.Expansions = t2.Expansions
	}
	if t2.Output.IsSet() {
		t1.Output = t2.Output
	}

	if t1.Vars == nil {
		t1.Vars = &Vars{}
	}
	if t1.Env == nil {
		t1.Env = &Vars{}
	}
	t1.Vars.Merge(t2.Vars)
	t1.Env.Merge(t2.Env)

	return t2.Tasks.Range(func(k string, v *Task) error {
		// We do a deep copy of the task struct here to ensure that no data can
		// be changed elsewhere once the taskfile is merged.
		task := v.DeepCopy()

		// Set the task to internal if EITHER the included task or the included
		// taskfile are marked as internal
		task.Internal = task.Internal || (includedTaskfile != nil && includedTaskfile.Internal)

		// Add namespaces to dependencies, commands and aliases
		for _, dep := range task.Deps {
			if dep != nil && dep.Task != "" {
				dep.Task = taskNameWithNamespace(dep.Task, namespaces...)
			}
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
				task.Aliases = append(task.Aliases, taskNameWithNamespace(task.Task, namespaceAlias))
				for _, alias := range v.Aliases {
					task.Aliases = append(task.Aliases, taskNameWithNamespace(alias, namespaceAlias))
				}
			}
		}

		// Add the task to the merged taskfile
		taskNameWithNamespace := taskNameWithNamespace(k, namespaces...)
		task.Task = taskNameWithNamespace
		t1.Tasks.Set(taskNameWithNamespace, task)

		return nil
	})
}

func taskNameWithNamespace(taskName string, namespaces ...string) string {
	if strings.HasPrefix(taskName, NamespaceSeparator) {
		return strings.TrimPrefix(taskName, NamespaceSeparator)
	}
	return strings.Join(append(namespaces, taskName), NamespaceSeparator)
}
