package ast

import (
	"fmt"

	"github.com/go-task/task/v3/errors"
)

// MergeOverride applies a local override: vars and env merge by key, tasks are
// replaced whole.
func (tf *Taskfile) MergeOverride(override *Taskfile) error {
	// An older version is fine, so bumping the shared Taskfile doesn't break
	// existing overrides.
	if override.Version.Major() != tf.Version.Major() || override.Version.GreaterThan(tf.Version) {
		return &errors.TaskfileVersionCheckError{
			URI:           override.Location,
			SchemaVersion: override.Version,
			Message:       fmt.Sprintf("must be a v%d version no newer than the main Taskfile (%s)", tf.Version.Major(), tf.Version),
		}
	}
	if tf.Vars == nil {
		tf.Vars = NewVars()
	}
	if tf.Env == nil {
		tf.Env = NewVars()
	}
	if tf.Tasks == nil {
		tf.Tasks = NewTasks()
	}
	tf.Vars.Merge(override.Vars, nil)
	tf.Env.Merge(override.Env, nil)
	for name, task := range override.Tasks.All(nil) {
		tf.Tasks.Set(name, task)
	}
	return nil
}
