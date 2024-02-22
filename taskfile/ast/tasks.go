package ast

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/omap"
)

// Tasks represents a group of tasks
type Tasks struct {
	omap.OrderedMap[string, *Task]
}

type MatchingTask struct {
	Task      *Task
	Wildcards []string
}

func (t *Tasks) FindMatchingTasks(call *Call) []*MatchingTask {
	if call == nil {
		return nil
	}
	var task *Task
	var matchingTasks []*MatchingTask
	// If there is a direct match, return it
	if task = t.OrderedMap.Get(call.Task); task != nil {
		matchingTasks = append(matchingTasks, &MatchingTask{Task: task, Wildcards: nil})
		return matchingTasks
	}
	// Attempt a wildcard match
	// For now, we can just nil check the task before each loop
	_ = t.Range(func(key string, value *Task) error {
		if match, wildcards := value.WildcardMatch(call.Task); match {
			matchingTasks = append(matchingTasks, &MatchingTask{
				Task:      value,
				Wildcards: wildcards,
			})
		}
		return nil
	})
	return matchingTasks
}

func (t1 *Tasks) Merge(t2 Tasks, include *Include) {
	_ = t2.Range(func(k string, v *Task) error {
		// We do a deep copy of the task struct here to ensure that no data can
		// be changed elsewhere once the taskfile is merged.
		task := v.DeepCopy()

		// Set the task to internal if EITHER the included task or the included
		// taskfile are marked as internal
		task.Internal = task.Internal || (include != nil && include.Internal)

		// Add namespaces to dependencies, commands and aliases
		for _, dep := range task.Deps {
			if dep != nil && dep.Task != "" {
				dep.Task = taskNameWithNamespace(dep.Task, include.Namespace)
			}
		}
		for _, cmd := range task.Cmds {
			if cmd != nil && cmd.Task != "" {
				cmd.Task = taskNameWithNamespace(cmd.Task, include.Namespace)
			}
		}
		for i, alias := range task.Aliases {
			task.Aliases[i] = taskNameWithNamespace(alias, include.Namespace)
		}
		// Add namespace aliases
		if include != nil {
			for _, namespaceAlias := range include.Aliases {
				task.Aliases = append(task.Aliases, taskNameWithNamespace(task.Task, namespaceAlias))
				for _, alias := range v.Aliases {
					task.Aliases = append(task.Aliases, taskNameWithNamespace(alias, namespaceAlias))
				}
			}
		}

		// Add the task to the merged taskfile
		taskNameWithNamespace := taskNameWithNamespace(k, include.Namespace)
		task.Task = taskNameWithNamespace
		t1.Set(taskNameWithNamespace, task)

		return nil
	})

	// If the included Taskfile has a default task and the parent namespace has
	// no task with a matching name, we can add an alias so that the user can
	// run the included Taskfile's default task without specifying its full
	// name. If the parent namespace has aliases, we add another alias for each
	// of them.
	if t2.Get("default") != nil && t1.Get(include.Namespace) == nil {
		defaultTaskName := fmt.Sprintf("%s:default", include.Namespace)
		t1.Get(defaultTaskName).Aliases = append(t1.Get(defaultTaskName).Aliases, include.Namespace)
		t1.Get(defaultTaskName).Aliases = append(t1.Get(defaultTaskName).Aliases, include.Aliases...)
	}
}

func (t *Tasks) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		tasks := omap.New[string, *Task]()
		if err := node.Decode(&tasks); err != nil {
			return err
		}

		// nolint: errcheck
		tasks.Range(func(name string, task *Task) error {
			// Set the task's name
			if task == nil {
				task = &Task{
					Task: name,
				}
			}
			task.Task = name

			// Set the task's location
			for _, keys := range node.Content {
				if keys.Value == name {
					task.Location = &Location{
						Line:   keys.Line,
						Column: keys.Column,
					}
				}
			}
			tasks.Set(name, task)
			return nil
		})

		*t = Tasks{
			OrderedMap: tasks,
		}
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into tasks", node.Line, node.ShortTag())
}

func taskNameWithNamespace(taskName string, namespace string) string {
	if strings.HasPrefix(taskName, NamespaceSeparator) {
		return strings.TrimPrefix(taskName, NamespaceSeparator)
	}
	return fmt.Sprintf("%s%s%s", namespace, NamespaceSeparator, taskName)
}
