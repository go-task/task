package ast

import "fmt"

// MergeOverride merges local variables and environment entries by key and
// replaces entire tasks. Existing keys keep their order; new keys are appended
// in declaration order. The main Taskfile's identity and settings are preserved.
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
		tf.Tasks.Set(name, task.DeepCopy())
	}
	return nil
}
