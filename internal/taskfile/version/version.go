package version

import (
	"github.com/Masterminds/semver"
)

var (
	v1  = mustVersion("1")
	v2  = mustVersion("2")
	v21 = mustVersion("2.1")
	v22 = mustVersion("2.2")
)

// IsV1 returns if is a given Taskfile version is version 1
func IsV1(v *semver.Constraints) bool {
	return v.Check(v1)
}

// IsV2 returns if is a given Taskfile version is at least version 2
func IsV2(v *semver.Constraints) bool {
	return v.Check(v2)
}

// IsV21 returns if is a given Taskfile version is at least version 2.1
func IsV21(v *semver.Constraints) bool {
	return v.Check(v21)
}

// IsV22 returns if is a given Taskfile version is at least version 2.2
func IsV22(v *semver.Constraints) bool {
	return v.Check(v22)
}

func mustVersion(s string) *semver.Version {
	v, err := semver.NewVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}
