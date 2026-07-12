package fsext

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-task/task/v3/errors"
)

var errFastGlobFallback = errors.New("fast glob fallback")

// FastRecursiveGlob expands simple literal-root recursive patterns. The
// boolean reports whether the pattern was handled or requires the full shell
// glob expander.
func FastRecursiveGlob(pattern string) ([]string, bool, error) {
	pattern = filepath.Clean(pattern)
	separator := string(os.PathSeparator)
	marker := separator + "**" + separator

	idx := strings.Index(pattern, marker)
	if idx == -1 || strings.Contains(pattern[idx+len(marker):], marker) {
		return nil, false, nil
	}

	root := pattern[:idx]
	namePattern := pattern[idx+len(marker):]
	if root == "" || namePattern == "" || strings.Contains(namePattern, separator) {
		return nil, false, nil
	}
	if strings.Contains(root, "**") || strings.ContainsAny(root, "*?[]{}") {
		return nil, false, nil
	}
	if strings.ContainsAny(namePattern, "{}") {
		return nil, false, nil
	}

	results := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if path == root {
			return errFastGlobFallback
		}

		if d.Type()&fs.ModeSymlink != 0 {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return errFastGlobFallback
			}
		}

		matched, err := filepath.Match(namePattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			results[path] = true
		}
		return nil
	})
	if errors.Is(err, errFastGlobFallback) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	return collectGlobKeys(results), true, nil
}

func collectGlobKeys(matches map[string]bool) []string {
	results := make([]string, 0, len(matches))
	for path := range matches {
		results = append(results, filepath.ToSlash(path))
	}
	sort.Strings(results)
	return results
}
