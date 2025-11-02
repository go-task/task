package ast

import (
	"cmp"
	"maps"
	"time"

	"github.com/Masterminds/semver/v3"
)

type TaskRC struct {
	Version     *semver.Version `yaml:"version"`
	Verbose     *bool           `yaml:"verbose"`
	Concurrency *int            `yaml:"concurrency"`
	Remote      Remote          `yaml:"remote"`
	Experiments map[string]int  `yaml:"experiments"`
}

type Remote struct {
	Insecure    *bool          `yaml:"insecure"`
	Offline     *bool          `yaml:"offline"`
	Timeout     *time.Duration `yaml:"timeout"`
	CacheExpiry *time.Duration `yaml:"cache-expiry"`
	Trust       []string       `yaml:"trust"`
}

// Merge combines the current TaskRC with another TaskRC, prioritizing non-nil fields from the other TaskRC.
func (t *TaskRC) Merge(other *TaskRC) {
	if other == nil {
		return
	}

	t.Version = cmp.Or(other.Version, t.Version)

	if t.Experiments == nil && other.Experiments != nil {
		t.Experiments = other.Experiments
	} else if t.Experiments != nil && other.Experiments != nil {
		maps.Copy(t.Experiments, other.Experiments)
	}

	// Merge Remote fields
	t.Remote.Insecure = cmp.Or(other.Remote.Insecure, t.Remote.Insecure)
	t.Remote.Offline = cmp.Or(other.Remote.Offline, t.Remote.Offline)
	t.Remote.Timeout = cmp.Or(other.Remote.Timeout, t.Remote.Timeout)
	t.Remote.CacheExpiry = cmp.Or(other.Remote.CacheExpiry, t.Remote.CacheExpiry)

	// Merge Trust lists - combine both lists with other's entries taking precedence
	// Remove duplicates by using a map
	if len(other.Remote.Trust) > 0 {
		seen := make(map[string]bool)
		merged := []string{}

		// Add other's hosts first
		for _, host := range other.Remote.Trust {
			if !seen[host] {
				seen[host] = true
				merged = append(merged, host)
			}
		}

		// Then add base's hosts that aren't duplicates
		for _, host := range t.Remote.Trust {
			if !seen[host] {
				seen[host] = true
				merged = append(merged, host)
			}
		}

		t.Remote.Trust = merged
	}

	t.Verbose = cmp.Or(other.Verbose, t.Verbose)
	t.Concurrency = cmp.Or(other.Concurrency, t.Concurrency)
}
