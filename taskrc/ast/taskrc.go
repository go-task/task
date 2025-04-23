package ast

import (
	"github.com/Masterminds/semver/v3"
	"time"
)

type TaskRC struct {
	Version     *semver.Version `yaml:"version"`
	Experiments map[string]int  `yaml:"experiments"`
	Remote      remote          `yaml:"remote"`
}

type remote struct {
	CacheExpiry *time.Duration `yaml:"cache-expiry"`
}
