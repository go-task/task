package ast

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/elliotchance/orderedmap/v3"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
)

type (
	// Tasks is an ordered map of task names to Tasks.
	Tasks struct {
		om    *orderedmap.OrderedMap[string, *Task]
		mutex sync.RWMutex
	}
	// A TaskElement is a key-value pair that is used for initializing a Tasks
	// structure.
	TaskElement orderedmap.Element[string, *Task]
	// MatchingTask represents a task that matches a given call. It includes the
	// task itself and a list of wildcards that were matched.
	MatchingTask struct {
		Task      *Task
		Wildcards []string
	}
)

// NewTasks creates a new instance of Tasks and initializes it with the provided
// set of elements, if any. The elements are added in the order they are passed.
func NewTasks(els ...*TaskElement) *Tasks {
	tasks := &Tasks{
		om: orderedmap.NewOrderedMap[string, *Task](),
	}
	for _, el := range els {
		tasks.Set(el.Key, el.Value)
	}
	return tasks
}

// Len returns the number of variables in the Tasks map.
func (tasks *Tasks) Len() int {
	if tasks == nil || tasks.om == nil {
		return 0
	}
	defer tasks.mutex.RUnlock()
	tasks.mutex.RLock()
	return tasks.om.Len()
}

// Get returns the value the the task with the provided key and a boolean that
// indicates if the value was found or not. If the value is not found, the
// returned task is a zero value and the bool is false.
func (tasks *Tasks) Get(key string) (*Task, bool) {
	if tasks == nil || tasks.om == nil {
		return &Task{}, false
	}
	defer tasks.mutex.RUnlock()
	tasks.mutex.RLock()
	return tasks.om.Get(key)
}

// Set sets the value of the task with the provided key to the provided value.
// If the task already exists, its value is updated. If the task does not exist,
// it is created.
func (tasks *Tasks) Set(key string, value *Task) bool {
	if tasks == nil {
		tasks = NewTasks()
	}
	if tasks.om == nil {
		tasks.om = orderedmap.NewOrderedMap[string, *Task]()
	}
	defer tasks.mutex.Unlock()
	tasks.mutex.Lock()
	return tasks.om.Set(key, value)
}

// Range calls the provided function for each task in the map. The function
// receives the task's key and value as arguments. If the function returns an
// error, the iteration stops and the error is returned.
func (tasks *Tasks) Range(f func(k string, v *Task) error) error {
	if tasks == nil || tasks.om == nil {
		return nil
	}
	for pair := tasks.om.Front(); pair != nil; pair = pair.Next() {
		if err := f(pair.Key, pair.Value); err != nil {
			return err
		}
	}
	return nil
}

// Keys returns a slice of all the keys in the Tasks map.
func (tasks *Tasks) Keys() []string {
	if tasks == nil {
		return nil
	}
	defer tasks.mutex.RUnlock()
	tasks.mutex.RLock()
	var keys []string
	for pair := tasks.om.Front(); pair != nil; pair = pair.Next() {
		keys = append(keys, pair.Key)
	}
	return keys
}

// Values returns a slice of all the values in the Tasks map.
func (tasks *Tasks) Values() []*Task {
	if tasks == nil {
		return nil
	}
	defer tasks.mutex.RUnlock()
	tasks.mutex.RLock()
	var values []*Task
	for pair := tasks.om.Front(); pair != nil; pair = pair.Next() {
		values = append(values, pair.Value)
	}
	return values
}

// FindMatchingTasks returns a list of tasks that match the given call. A task
// matches a call if its name is equal to the call's task name or if it matches
// a wildcard pattern. The function returns a list of MatchingTask structs, each
// containing a task and a list of wildcards that were matched.
func (t *Tasks) FindMatchingTasks(call *Call) []*MatchingTask {
	if call == nil {
		return nil
	}
	var matchingTasks []*MatchingTask
	// If there is a direct match, return it
	if task, ok := t.Get(call.Task); ok {
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

func (t1 *Tasks) Merge(t2 *Tasks, include *Include, includedTaskfileVars *Vars) error {
	defer t2.mutex.RUnlock()
	t2.mutex.RLock()
	err := t2.Range(func(name string, v *Task) error {
		// We do a deep copy of the task struct here to ensure that no data can
		// be changed elsewhere once the taskfile is merged.
		task := v.DeepCopy()
		// Set the task to internal if EITHER the included task or the included
		// taskfile are marked as internal
		task.Internal = task.Internal || (include != nil && include.Internal)
		taskName := name

		// if the task is in the exclude list, don't add it to the merged taskfile and early return
		if slices.Contains(include.Excludes, name) {
			return nil
		}

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
				task.IncludeVars = NewVars()
			}
			task.IncludeVars.Merge(include.Vars, nil)
			task.IncludedTaskfileVars = includedTaskfileVars.DeepCopy()
		}

		if _, ok := t1.Get(taskName); ok {
			return &errors.TaskNameFlattenConflictError{
				TaskName: taskName,
				Include:  include.Namespace,
			}
		}
		// Add the task to the merged taskfile
		t1.Set(taskName, task)

		return nil
	})

	// If the included Taskfile has a default task, is not flattened and the
	// parent namespace has no task with a matching name, we can add an alias so
	// that the user can run the included Taskfile's default task without
	// specifying its full name. If the parent namespace has aliases, we add
	// another alias for each of them.
	_, t2DefaultExists := t2.Get("default")
	_, t1NamespaceExists := t1.Get(include.Namespace)
	if t2DefaultExists && !t1NamespaceExists && !include.Flatten {
		defaultTaskName := fmt.Sprintf("%s:default", include.Namespace)
		t1DefaultTask, ok := t1.Get(defaultTaskName)
		if ok {
			t1DefaultTask.Aliases = append(t1DefaultTask.Aliases, include.Namespace)
			t1DefaultTask.Aliases = slices.Concat(t1DefaultTask.Aliases, include.Aliases)
		}
	}

	return err
}

func (t *Tasks) UnmarshalYAML(node *yaml.Node) error {
	if t == nil || t.om == nil {
		*t = *NewTasks()
	}
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE: orderedmap does not have an unmarshaler, so we have to decode
		// the map manually. We increment over 2 values at a time and assign
		// them as a key-value pair.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Decode the value node into a Task struct
			var v Task
			if err := valueNode.Decode(&v); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}

			// Set the task name and location
			v.Task = keyNode.Value
			v.Location = &Location{
				Line:   keyNode.Line,
				Column: keyNode.Column,
			}

			// Add the task to the ordered map
			t.Set(keyNode.Value, &v)
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
