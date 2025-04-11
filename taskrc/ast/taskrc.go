package ast

import "github.com/Masterminds/semver/v3"

type TaskRC struct {
	Version     *semver.Version `yaml:"version"`
	Experiments map[string]int  `yaml:"experiments"`
}
