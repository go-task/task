package taskfile

import (
	"fmt"
)

// Merge merges the second Taskfile into the first
func Merge(t1, t2 *Taskfile) error {
	if t1.Version != t2.Version {
		return fmt.Errorf(`Taskfiles versions should match. First is "%s" but second is "%s"`, t1.Version, t2.Version)
	}

	if t2.Expansions != 0 && t2.Expansions != 2 {
		t1.Expansions = t2.Expansions
	}
	if t2.Output != "" {
		t1.Output = t2.Output
	}
	for k, v := range t2.Vars {
		t1.Vars[k] = v
	}
	for k, v := range t2.Tasks {
		t1.Tasks[k] = v
	}

	return nil
}
