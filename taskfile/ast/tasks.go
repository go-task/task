package ast

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
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

func (t1 *Tasks) Merge(t2 Tasks, include *Include, includedTaskfileVars *Vars) error {
	err := t2.Range(func(name string, v *Task) error {
		// We do a deep copy of the task struct here to ensure that no data can
		// be changed elsewhere once the taskfile is merged.
		task := v.DeepCopy()
		// Set the task to internal if EITHER the included task or the included
		// taskfile are marked as internal
		task.Internal = task.Internal || (include != nil && include.Internal)
		taskName := name
		if !include.Flatten {
			// Add namespaces to task dependencies
			for _, dep := range task.Deps {
				if dep != nil && dep.Task != "" {
					dep.Task = taskNameWithNamespace(dep.Task, include.Namespace)
				}
			}

			// Add namespaces to task commands
			for _, cmd := range task.Cmds {
				if cmd != nil && cmd.Task != "" {
					cmd.Task = taskNameWithNamespace(cmd.Task, include.Namespace)
				}
			}

			// Add namespaces to task aliases
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

			taskName = taskNameWithNamespace(name, include.Namespace)
			task.Namespace = include.Namespace
			task.Task = taskName
		}

		if include.AdvancedImport {
			task.Dir = filepathext.SmartJoin(include.Dir, task.Dir)
			if task.IncludeVars == nil {
				task.IncludeVars = &Vars{}
			}
			task.IncludeVars.Merge(include.Vars, nil)
			task.IncludedTaskfileVars = includedTaskfileVars.DeepCopy()
		}

		if t1.Get(taskName) != nil {
			return &errors.TaskNameFlattenConflictError{
				TaskName: taskName,
				Include:  include.Namespace,
			}
		}
		// Add the task to the merged taskfile
		t1.Set(taskName, task)

		return nil
	})

	// If the included Taskfile has a default task, being not flattened and the parent namespace has
	// no task with a matching name, we can add an alias so that the user can
	// run the included Taskfile's default task without specifying its full
	// name. If the parent namespace has aliases, we add another alias for each
	// of them.
	if t2.Get("default") != nil && t1.Get(include.Namespace) == nil && !include.Flatten {
		defaultTaskName := fmt.Sprintf("%s:default", include.Namespace)
		t1.Get(defaultTaskName).Aliases = append(t1.Get(defaultTaskName).Aliases, include.Namespace)
		t1.Get(defaultTaskName).Aliases = slices.Concat(t1.Get(defaultTaskName).Aliases, include.Aliases)
	}
	return err
}

func (t *Tasks) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		tasks := omap.New[string, *Task]()
		if err := node.Decode(&tasks); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
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

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("tasks")
}

func taskNameWithNamespace(taskName string, namespace string) string {
	if strings.HasPrefix(taskName, NamespaceSeparator) {
		return strings.TrimPrefix(taskName, NamespaceSeparator)
	}
	return fmt.Sprintf("%s%s%s", namespace, NamespaceSeparator, taskName)
}
