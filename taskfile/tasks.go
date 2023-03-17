package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Tasks represents a group of tasks
type Tasks map[string]*Task

func (t *Tasks) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		tasks := map[string]*Task{}
		if err := node.Decode(tasks); err != nil {
			return err
		}

		for name := range tasks {
			// Set the task's name
			if tasks[name] == nil {
				tasks[name] = &Task{
					Task: name,
				}
			}
			tasks[name].Task = name

			// Set the task's location
			for _, keys := range node.Content {
				if keys.Value == name {
					tasks[name].Location = &Location{
						Line:   keys.Line,
						Column: keys.Column,
					}
				}
			}
		}

		*t = Tasks(tasks)
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into tasks", node.Line, node.ShortTag())
}
