package ast

import "github.com/Masterminds/semver/v3"

type TaskRC struct {
	Version     *semver.Version `yaml:"version"`
	Experiments map[string]int  `yaml:"experiments"`
}

// Merge combines the current TaskRC with another TaskRC, prioritizing non-nil fields from the other TaskRC.
func (t *TaskRC) Merge(other *TaskRC) {
	if other == nil {
		return
	}
	if t.Version == nil && other.Version != nil {
		t.Version = other.Version
	}
	if t.Experiments == nil && other.Experiments != nil {
		t.Experiments = other.Experiments
	} else if t.Experiments != nil && other.Experiments != nil {
		for k, v := range other.Experiments {
			t.Experiments[k] = v
		}
	}
}
