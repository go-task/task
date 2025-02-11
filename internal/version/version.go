package version

import (
	"fmt"
	"runtime/debug"
)

var (
	version = ""
	sum     = ""
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" || info.Main.Version == "" {
		version = "unknown"
	} else {
		if version == "" {
			version = info.Main.Version
		}
		if sum == "" {
			sum = info.Main.Sum
		}
	}
}

func GetVersion() string {
	return version
}

func GetVersionWithSum() string {
	return fmt.Sprintf("%s (%s)", version, sum)
}
