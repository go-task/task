package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/orderedmap"
)

// Tasks represents a group of tasks
type Tasks struct {
	orderedmap.OrderedMap[string, *Task]
}

func (t *Tasks) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		tasks := orderedmap.New[string, *Task]()
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
