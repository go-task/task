package version

import (
	"fmt"
	"runtime/debug"
)

var version = ""

func GetVersion(withSum bool) string {
	if version != "" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "unknown"
	}

	ver := info.Main.Version
	if info.Main.Sum != "" && withSum {
		ver += fmt.Sprintf(" (%s)", info.Main.Sum)
	}
	return ver
}
