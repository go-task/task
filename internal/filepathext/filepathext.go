package filepathext

import (
	"os"
	"path/filepath"
)

// SmartJoin joins two paths, but only if the second is not already an
// absolute path.
func SmartJoin(a, b string) string {
	if filepath.IsAbs(b) {
		return b
	}
	return filepath.Join(a, b)
}

// TryAbsToRel tries to convert an absolute path to relative based on the
// process working directory. If it can't, it returns the absolute path.
func TryAbsToRel(abs string) string {
	wd, err := os.Getwd()
	if err != nil {
		return abs
	}

	rel, err := filepath.Rel(wd, abs)
	if err != nil {
		return abs
	}

	return rel
}
