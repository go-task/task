package version

import (
	"runtime/debug"
)

var (
	version = "unknown"
	sum     = ""
)

func Init() {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return
	}

	version = info.Main.Version
	sum = info.Main.Sum
}

func GetVersion() string {
	return version
}

func GetVersionWithSum() string {
	result := version
	if sum != "" {
		result += " (" + sum + ")"
	}
	return result
}
