package version

import (
	"github.com/Masterminds/semver"
)

var (
	v1 = mustVersion("1")
	v2 = mustVersion("2")

	isV1  = mustConstraint("= 1")
	isV2  = mustConstraint(">= 2")
	isV21 = mustConstraint(">= 2.1")
)

// IsV1 returns if is a given Taskfile version is version 1
func IsV1(v *semver.Version) bool {
	return isV1.Check(v)
}

// IsV2 returns if is a given Taskfile version is at least version 2
func IsV2(v *semver.Version) bool {
	return isV2.Check(v)
}

// IsV21 returns if is a given Taskfile version is at least version 2
func IsV21(v *semver.Version) bool {
	return isV21.Check(v)
}

func mustVersion(s string) *semver.Version {
	v, err := semver.NewVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

func mustConstraint(s string) *semver.Constraints {
	c, err := semver.NewConstraint(s)
	if err != nil {
		panic(err)
	}
	return c
}
