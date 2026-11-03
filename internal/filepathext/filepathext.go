package filepathext

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

// SmartJoin joins two paths, but only if the second is not already an
// absolute path.
func SmartJoin(a, b string) string {
	if IsAbs(b) {
		return b
	}
	return filepath.Join(a, b)
}

func IsAbs(path string) bool {
	// NOTE(@andreynering): If the path contains any if the special
	// variables that we know are absolute, return true.
	if isSpecialDir(path) {
		return true
	}

	return filepath.IsAbs(path)
}

var (
	templateActionRE = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	// Use a Unicode-aware boundary so longer variable names remain relative.
	specialDirRE = regexp.MustCompile(`\.(ROOT_DIR|TASKFILE_DIR|USER_WORKING_DIR)([^\pL\p{Nd}_]|$)`)
)

func isSpecialDir(dir string) bool {
	return slices.ContainsFunc(templateActionRE.FindAllString(dir, -1), specialDirRE.MatchString)
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

// IsExtOnly checks whether path points to a file with no name but with
// an extension, i.e. ".yaml"
func IsExtOnly(path string) bool {
	return filepath.Base(path) == filepath.Ext(path)
}
