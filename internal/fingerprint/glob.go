package fingerprint

import (
	"os"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
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

	// Optimize simple globstar patterns (most common case) by using doublestar library
	// instead of shell expansion, which is much faster for large directory trees
	if strings.Contains(g, "**") && !hasShellFeatures(g) {
		matches, err := doublestar.FilepathGlob(g, doublestar.WithFilesOnly())
		if err != nil {
			return nil, err
		}
		return matches, nil
	}

	// Fall back to shell expansion for complex patterns or non-globstar patterns
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

// hasShellFeatures checks if pattern needs shell expansion (env vars, tilde, braces)
func hasShellFeatures(pattern string) bool {
	return strings.Contains(pattern, "$") || strings.Contains(pattern, "~") ||
		(strings.Contains(pattern, "{") && strings.Contains(pattern, "}"))
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
