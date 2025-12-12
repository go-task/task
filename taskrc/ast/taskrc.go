package ast

import (
	"cmp"
	"maps"
	"slices"
	"time"

	"github.com/Masterminds/semver/v3"
)

type TaskRC struct {
	Version      *semver.Version `yaml:"version"`
	Verbose      *bool           `yaml:"verbose"`
	Color        *bool           `yaml:"color"`
	DisableFuzzy *bool           `yaml:"disable-fuzzy"`
	Concurrency  *int            `yaml:"concurrency"`
	Remote       Remote          `yaml:"remote"`
	Failfast     bool            `yaml:"failfast"`
	Experiments  map[string]int  `yaml:"experiments"`
}

type Remote struct {
	Insecure     *bool          `yaml:"insecure"`
	Offline      *bool          `yaml:"offline"`
	Timeout      *time.Duration `yaml:"timeout"`
	CacheExpiry  *time.Duration `yaml:"cache-expiry"`
	CacheDir     *string        `yaml:"cache-dir"`
	TrustedHosts []string       `yaml:"trusted-hosts"`
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
	t.Remote.CacheDir = cmp.Or(other.Remote.CacheDir, t.Remote.CacheDir)
	if len(other.Remote.TrustedHosts) > 0 {
		merged := slices.Concat(other.Remote.TrustedHosts, t.Remote.TrustedHosts)
		slices.Sort(merged)
		t.Remote.TrustedHosts = slices.Compact(merged)
	}

	t.Verbose = cmp.Or(other.Verbose, t.Verbose)
	t.Color = cmp.Or(other.Color, t.Color)
	t.DisableFuzzy = cmp.Or(other.DisableFuzzy, t.DisableFuzzy)
	t.Concurrency = cmp.Or(other.Concurrency, t.Concurrency)
	t.Failfast = cmp.Or(other.Failfast, t.Failfast)
}
