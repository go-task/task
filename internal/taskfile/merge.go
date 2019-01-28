package taskfile

import (
	"fmt"
)

// Merge merges the second Taskfile into the first
func Merge(inc *Include, t1, t2 *Taskfile, namespaces ...string) error {
	if t1.Version != t2.Version {
		return fmt.Errorf(
			`Taskfiles versions should match. First is "%s" but second is "%s"`,
			t1.Version, t2.Version,
		)
	}

	if t2.Expansions != 0 && t2.Expansions != 2 {
		t1.Expansions = t2.Expansions
	}
	if t2.Output != "" {
		t1.Output = t2.Output
	}
	if t1.Includes == nil {
		t1.Includes = Includes{}
	}
	if t1.Vars == nil {
		t1.Vars = Vars{}
	}
	if t1.Tasks == nil {
		t1.Tasks = Tasks{}
	}
	for k, v := range t2.Includes {
		t1.Includes[k] = v
	}
	for k, v := range t2.Vars {
		t1.Vars[k] = v
	}
	for k, v := range t2.Tasks {
		if inc != nil {
			v.Hidden = inc.Hidden
			v.direct = inc.Direct
		}
		tasks := v.ApplyNamespace(k, namespaces...)
		for _, task := range tasks {
			if _, ok := t1.Tasks[task.Task]; !ok {
				t1.Tasks[task.Task] = task
			}
		}
	}

	return nil
}
