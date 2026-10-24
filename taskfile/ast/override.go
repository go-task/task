package ast

import "fmt"

// MergeOverride applies a local override: vars and env merge by key, tasks are
// replaced whole.
func (tf *Taskfile) MergeOverride(override *Taskfile) error {
	if !tf.Version.Equal(override.Version) {
		return fmt.Errorf("task: Taskfile override %q has version %s, but %q has version %s",
			override.Location, override.Version, tf.Location, tf.Version)
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
