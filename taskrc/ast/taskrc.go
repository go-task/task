package ast

import (
	"maps"
	"time"

	"github.com/Masterminds/semver/v3"
)

type TaskRC struct {
	Version     *semver.Version `yaml:"version"`
	Experiments map[string]int  `yaml:"experiments"`
	Remote      Remote          `yaml:"remote"`
	Verbose     *bool           `yaml:"verbose"`
	Concurrency *int            `yaml:"concurrency"`
}

type Remote struct {
	Insecure    *bool          `yaml:"insecure"`
	Offline     *bool          `yaml:"offline"`
	Timeout     *time.Duration `yaml:"timeout"`
	CacheExpiry *time.Duration `yaml:"cache-expiry"`
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
		maps.Copy(t.Experiments, other.Experiments)
	}

	// Merge Remote fields
	if other.Remote.Insecure != nil {
		t.Remote.Insecure = other.Remote.Insecure
	}
	if other.Remote.Offline != nil {
		t.Remote.Offline = other.Remote.Offline
	}

	if other.Remote.Timeout != nil {
		t.Remote.Timeout = other.Remote.Timeout
	}

	if other.Remote.CacheExpiry != nil {
		t.Remote.CacheExpiry = other.Remote.CacheExpiry
	}

	if other.Verbose != nil {
		t.Verbose = other.Verbose
	}

	if other.Concurrency != nil {
		t.Concurrency = other.Concurrency
	}
}
