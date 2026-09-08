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
	Silent       *bool           `yaml:"silent"`
	Color        *bool           `yaml:"color"`
	DisableFuzzy *bool           `yaml:"disable-fuzzy"`
	Concurrency  *int            `yaml:"concurrency"`
	Interactive  *bool           `yaml:"interactive"`
	Remote       Remote          `yaml:"remote"`
	Failfast     bool            `yaml:"failfast"`
	TempDir      *string         `yaml:"temp-dir"`
	Experiments  map[string]int  `yaml:"experiments"`
}

type Remote struct {
	Insecure     *bool           `yaml:"insecure"`
	Offline      *bool           `yaml:"offline"`
	Timeout      *time.Duration  `yaml:"timeout"`
	CacheExpiry  *time.Duration  `yaml:"cache-expiry"`
	CacheDir     *string         `yaml:"cache-dir"`
	TrustedHosts []string        `yaml:"trusted-hosts"`
	Headers      []RemoteHeaders `yaml:"headers"`
	CACert       *string         `yaml:"cacert"`
	Cert         *string         `yaml:"cert"`
	CertKey      *string         `yaml:"cert-key"`
}

// RemoteHeaders configures HTTP headers for a single host.
type RemoteHeaders struct {
	Host    string            `yaml:"host"`
	Headers map[string]string `yaml:"headers"`
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
	t.Remote.Headers = mergeHeaders(t.Remote.Headers, other.Remote.Headers)
	t.Remote.CACert = cmp.Or(other.Remote.CACert, t.Remote.CACert)
	t.Remote.Cert = cmp.Or(other.Remote.Cert, t.Remote.Cert)
	t.Remote.CertKey = cmp.Or(other.Remote.CertKey, t.Remote.CertKey)

	t.Verbose = cmp.Or(other.Verbose, t.Verbose)
	t.Silent = cmp.Or(other.Silent, t.Silent)
	t.Color = cmp.Or(other.Color, t.Color)
	t.DisableFuzzy = cmp.Or(other.DisableFuzzy, t.DisableFuzzy)
	t.Concurrency = cmp.Or(other.Concurrency, t.Concurrency)
	t.Interactive = cmp.Or(other.Interactive, t.Interactive)
	t.Failfast = cmp.Or(other.Failfast, t.Failfast)
	t.TempDir = cmp.Or(other.TempDir, t.TempDir)
}

// Replace each host's headers as a whole so closer config files can drop headers.
func mergeHeaders(base, other []RemoteHeaders) []RemoteHeaders {
	if len(other) == 0 {
		return base
	}
	byHost := make(map[string]RemoteHeaders, len(base)+len(other))
	for _, entry := range slices.Concat(base, other) {
		byHost[entry.Host] = entry
	}
	merged := slices.Collect(maps.Values(byHost))
	slices.SortFunc(merged, func(a, b RemoteHeaders) int {
		return cmp.Compare(a.Host, b.Host)
	})
	return merged
}
