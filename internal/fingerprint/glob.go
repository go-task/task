package fingerprint

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/mattn/go-zglob"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile/ast"
)

func Globs(dir string, gs []*ast.Glob) ([]string, error) {
	return globs(dir, gs, false)
}

func GlobsDirs(dir string, gs []*ast.Glob) ([]string, error) {
	return globs(dir, gs, true)
}

func globs(dir string, globs []*ast.Glob, dirOnly bool) ([]string, error) {
	resultMap := make(map[string]bool)
	for _, g := range globs {
		matches, err := glob(dir, g.Glob, dirOnly)
		if err != nil {
			continue
		}
		for _, match := range matches {
			resultMap[match] = !g.Negate
		}
	}
	return collectKeys(resultMap), nil
}

func glob(dir string, g string, dirOnly bool) ([]string, error) {
	g = filepathext.SmartJoin(dir, g)

	g, err := execext.Expand(g)
	if err != nil {
		return nil, err
	}

	fs, err := zglob.GlobFollowSymlinks(g)
	if err != nil {
		return nil, err
	}

	results := make(map[string]bool, len(fs))

	for _, f := range fs {
		info, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		if dirOnly {
			if info.IsDir() {
				results[f] = true
			} else {
				results[filepath.Dir(f)] = true
			}
			continue
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
