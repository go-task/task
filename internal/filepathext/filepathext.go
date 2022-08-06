package filepathext

import (
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
