package version

import (
	_ "embed"
	"runtime/debug"
	"strings"
)

var (
	//go:embed version.txt
	version string
	commit  string
	dirty   bool
)

func init() {
	version = strings.TrimSpace(version)
	// Attempt to get build info from the Go runtime. We only use this if not
	// built from a tagged version.
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version == "(devel)" {
		commit = getCommit(info)
		dirty = getDirty(info)
	}
}

func getDirty(info *debug.BuildInfo) bool {
	for _, setting := range info.Settings {
		if setting.Key == "vcs.modified" {
			return setting.Value == "true"
		}
	}
	return false
}

func getCommit(info *debug.BuildInfo) string {
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value[:7]
		}
	}
	return ""
}

// GetVersion returns the version of Task. By default, this is retrieved from
// the embedded version.txt file which is kept up-to-date by our release script.
// However, it can also be overridden at build time using:
// -ldflags="-X 'github.com/go-task/task/v3/internal/version.version=vX.X.X'".
func GetVersion() string {
	return version
}

// GetVersionWithBuildInfo is the same as [GetVersion], but it also includes
// the commit hash and dirty status if available. This will only work when built
// within inside of a Git checkout.
func GetVersionWithBuildInfo() string {
	var buildMetadata []string
	if commit != "" {
		buildMetadata = append(buildMetadata, commit)
	}
	if dirty {
		buildMetadata = append(buildMetadata, "dirty")
	}
	if len(buildMetadata) > 0 {
		return version + "+" + strings.Join(buildMetadata, ".")
	}
	return version
}
