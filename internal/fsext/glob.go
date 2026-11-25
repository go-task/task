package fsext

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// FastRecursiveGlob expands literal-root recursive patterns. The boolean
// reports whether the pattern was handled or requires the full shell glob
// expander.
//
// Unlike the shell globstar it replaces, symbolic links to directories are
// walked, so sources keep resolving through them. Loops are cut by tracking
// the directories already visited.
func FastRecursiveGlob(pattern string) ([]string, bool, error) {
	pattern = filepath.Clean(pattern)
	separator := string(os.PathSeparator)
	marker := separator + "**" + separator

	idx := strings.Index(pattern, marker)
	if idx == -1 || strings.Contains(pattern[idx+len(marker):], marker) {
		return nil, false, nil
	}

	root := pattern[:idx]
	suffix := pattern[idx+len(marker):]
	if root == "" || suffix == "" {
		return nil, false, nil
	}
	if strings.Contains(root, "**") || strings.ContainsAny(root, "*?[]{}") {
		return nil, false, nil
	}
	suffixParts := strings.Split(suffix, separator)
	for _, part := range suffixParts {
		if part == "" || strings.ContainsAny(part, "{}") {
			return nil, false, nil
		}
	}

	rootInfo, err := os.Stat(root)
	if err != nil {
		return nil, true, err
	}
	rootLink, err := os.Lstat(root)
	if err != nil {
		return nil, true, err
	}
	if !rootInfo.IsDir() || rootLink.Mode()&fs.ModeSymlink != 0 {
		// A non-directory root has nothing to recurse into, and a symlinked
		// one keeps the shell expander's semantics for the root itself.
		return nil, false, nil
	}

	var results []string
	visited := make(map[string]struct{})
	if err := walkRecursiveGlob(root, root, suffixParts, visited, &results); err != nil {
		return nil, true, err
	}
	slices.Sort(results)
	return results, true, nil
}

// walkRecursiveGlob matches suffixParts against the files below dir, following
// symbolic links to directories. A directory is only visited once, so a link
// cycle terminates instead of looping forever.
func walkRecursiveGlob(root, dir string, suffixParts []string, visited map[string]struct{}, results *[]string) error {
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	if _, seen := visited[canonical]; seen {
		return nil
	}
	visited[canonical] = struct{}{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if err := walkRecursiveGlob(root, path, suffixParts, visited, results); err != nil {
				return err
			}
			continue
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := walkRecursiveGlob(root, path, suffixParts, visited, results); err != nil {
					return err
				}
				continue
			}
		}

		if matchesSuffix(root, path, suffixParts) {
			*results = append(*results, filepath.ToSlash(path))
		}
	}
	return nil
}

// matchesSuffix reports whether the path below root matches the suffix of a
// "**" pattern: the globstar consumes any number of directories, so the
// suffix parts only need to match the deepest path parts.
func matchesSuffix(root, path string, suffixParts []string) bool {
	relative := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(root)+"/")
	parts := strings.Split(relative, "/")
	if len(parts) < len(suffixParts) {
		return false
	}
	parts = parts[len(parts)-len(suffixParts):]
	for i, part := range parts {
		matched, err := filepath.Match(suffixParts[i], part)
		if err != nil || !matched {
			return false
		}
	}
	return true
}
