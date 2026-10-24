package taskfile

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/taskfile/ast"
)

var defaultOverrides = []string{
	"Taskfile.override.yml",
	"taskfile.override.yml",
	"Taskfile.override.yaml",
	"taskfile.override.yaml",
}

func readAutomaticOverride(dir string, tf *ast.Taskfile) error {
	for _, name := range defaultOverrides {
		path := filepath.Join(dir, name)
		// Check existence separately so a dangling symlink is an error too.
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("task: Failed to read Taskfile override: %w", err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("task: Failed to read Taskfile override: %w", err)
		}
		override, err := decodeTaskfile(b, path, true)
		if err != nil {
			return err
		}
		return tf.MergeOverride(override)
	}
	return nil
}

// taskfileOverride restricts the root fields while using the normal Taskfile
// decoders for their values.
type taskfileOverride ast.Taskfile

func (tf *taskfileOverride) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("taskfile override")
	}
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		switch key.Value {
		case "version", "vars", "env", "tasks":
		default:
			return errors.NewTaskfileDecodeError(nil, key).
				WithMessage("field %q is not allowed in a Taskfile override", key.Value)
		}
	}
	return (*ast.Taskfile)(tf).UnmarshalYAML(node)
}
