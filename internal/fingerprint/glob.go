package fingerprint

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile/ast"
)

func Globs(dir string, globs []*ast.Glob) ([]string, error) {
	resultMap := make(map[string]bool)
	for _, g := range globs {
		matches, err := glob(dir, g.Glob)
		if err != nil {
			continue
		}
		for _, match := range matches {
			resultMap[match] = !g.Negate
		}
	}
	return collectKeys(resultMap), nil
}

func glob(dir string, g string) ([]string, error) {
	g = filepathext.SmartJoin(dir, g)

	fs, err := execext.ExpandFields(g)
	if err != nil {
		return nil, err
	}

	results := make(map[string]bool, len(fs))

	for _, f := range fs {
		info, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		results[f] = true
	}
	return collectKeys(results), nil
}

func collectKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// errStop is a sentinel error used to short-circuit filepath.WalkDir.
var errStop = errors.New("stop walk")

// walkGlobFiles walks files matching the given globs and calls visit for each
// matched file's FileInfo. If visit returns errStop, walking stops immediately
// (this is not treated as an error). For negated globs, it falls back to the
// standard Globs approach since negation requires full enumeration.
// Returns (fallback=true, nil) when negated globs are present so the caller
// can handle the fallback.
func walkGlobFiles(dir string, globs []*ast.Glob, visit func(path string, info fs.FileInfo) error) (negated bool, err error) {
	for _, g := range globs {
		if g.Negate {
			return true, nil
		}
	}

	for _, g := range globs {
		pattern := filepathext.SmartJoin(dir, g.Glob)
		base, _ := splitGlobPattern(pattern)

		walkErr := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !matchesPattern(path, base, pattern) {
				return nil
			}
			info, infoErr := d.Info()
			if infoErr != nil {
				return nil
			}
			return visit(path, info)
		})
		if walkErr != nil && !errors.Is(walkErr, errStop) {
			continue
		}
		if errors.Is(walkErr, errStop) {
			return false, nil
		}
	}
	return false, nil
}

// matchesPattern checks whether a file path matches a glob pattern.
// It tries filepath.Match first, then falls back to matchesGlobStar
// for ** patterns that filepath.Match doesn't support.
func matchesPattern(path, base, pattern string) bool {
	_, globSuffix := splitGlobPattern(pattern)

	// Try matching the full path against the suffix.
	if matched, err := filepath.Match(globSuffix, path); err == nil && matched {
		return true
	}

	// Try matching the relative path from base.
	if rel, err := filepath.Rel(base, path); err == nil {
		if matched, err := filepath.Match(globSuffix, rel); err == nil && matched {
			return true
		}
	}

	// Fall back to ** matching against the full pattern.
	return matchesGlobStar(path, pattern)
}

// anyGlobNewerThan checks if any file matching the given globs has a modification
// time newer than referenceTime. It walks each glob's base directory and checks
// timestamps inline, short-circuiting as soon as a newer file is found.
func anyGlobNewerThan(dir string, globs []*ast.Glob, referenceTime time.Time) (bool, error) {
	found := false
	negated, err := walkGlobFiles(dir, globs, func(_ string, info fs.FileInfo) error {
		if info.ModTime().After(referenceTime) {
			found = true
			return errStop
		}
		return nil
	})
	if negated {
		sources, err := Globs(dir, globs)
		if err != nil {
			return false, err
		}
		return anyFileNewerThan(sources, referenceTime)
	}
	return found, err
}

// GlobsMaxTime returns the maximum modification time among all files matching
// the given globs. It walks directories directly to avoid the expensive full
// glob expansion via execext.ExpandFields.
func GlobsMaxTime(dir string, globs []*ast.Glob) (time.Time, error) {
	var maxT time.Time
	negated, err := walkGlobFiles(dir, globs, func(_ string, info fs.FileInfo) error {
		if info.ModTime().After(maxT) {
			maxT = info.ModTime()
		}
		return nil
	})
	if negated {
		sources, err := Globs(dir, globs)
		if err != nil {
			return time.Time{}, err
		}
		return getMaxTime(sources...)
	}
	return maxT, err
}

// splitGlobPattern splits a glob pattern into a concrete base directory
// and the remaining glob suffix. For example:
//
//	"/home/user/project/gqlgen/**/*.gql" → ("/home/user/project/gqlgen", "**/*.gql")
//	"./src/**/*.gql" → ("./src", "**/*.gql")
func splitGlobPattern(pattern string) (base, suffix string) {
	dir := pattern
	for {
		if dir == "." || dir == "/" {
			return dir, pattern
		}
		base := filepath.Base(dir)
		if strings.ContainsAny(base, "*?[{") {
			dir = filepath.Dir(dir)
			continue
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if !strings.ContainsAny(parent, "*?[{") {
			return dir, strings.TrimPrefix(strings.TrimPrefix(pattern, dir), string(filepath.Separator))
		}
		dir = parent
	}
	return ".", pattern
}

// matchesGlobStar checks if a file path matches a pattern containing **.
// It handles the common cases of "base/**/*.ext" and "base/**".
func matchesGlobStar(filePath, pattern string) bool {
	if idx := strings.Index(pattern, "/**/"); idx >= 0 {
		base := pattern[:idx]
		suffix := pattern[idx+4:] // after "/**/"

		if !strings.HasPrefix(filePath, base+"/") && filePath != base {
			return false
		}

		if !strings.ContainsAny(suffix, "*?[{") {
			return filepath.Base(filePath) == suffix
		}

		matched, err := filepath.Match(suffix, filepath.Base(filePath))
		return err == nil && matched
	}

	if strings.HasSuffix(pattern, "/**") {
		base := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(filePath, base+"/")
	}

	return false
}
