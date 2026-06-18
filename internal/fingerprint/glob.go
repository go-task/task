package fingerprint

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile/ast"
)

var errFastGlobFallback = errors.New("fast glob fallback")

func Globs(dir string, globs []*ast.Glob, useGitignore bool) ([]string, error) {
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

	if useGitignore {
		resultMap = filterGitignored(resultMap, dir)
	}

	return collectKeys(resultMap), nil
}

func glob(dir string, g string) ([]string, error) {
	g = filepathext.SmartJoin(dir, g)

	if results, ok, err := fastRecursiveGlob(g); ok {
		return results, err
	}

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

func fastRecursiveGlob(pattern string) ([]string, bool, error) {
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
		if !matched {
			return nil
		}
		results[path] = true
		return nil
	})
	if errors.Is(err, errFastGlobFallback) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	return collectKeys(results), true, nil
}

func collectKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			// Normalize path separators for consistent sorting across platforms
			keys = append(keys, filepath.ToSlash(k))
		}
	}
	sort.Strings(keys)
	return keys
}
