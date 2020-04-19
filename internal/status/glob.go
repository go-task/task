package status

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-task/task/v2/internal/execext"

	"github.com/mattn/go-zglob"
)

// Matcher represents a matcher to sources
type Matcher struct {
	dir      string
	includes []string
	excludes []string
}

// NewMatcher creates a Matcher
func NewMatcher(dir string, globs []string) (*Matcher, error) {
	var sm = Matcher{
		dir:      dir,
		includes: make([]string, 0, len(globs)),
		excludes: make([]string, 0, len(globs)/2+1),
	}

	for _, g := range globs {
		isExclude := strings.HasPrefix(g, "!")
		if isExclude {
			g = g[1:]
		}
		g, err := execext.Expand(g)
		if err != nil {
			return nil, err
		}

		if isExclude {
			sm.excludes = append(sm.excludes, g)
		} else {
			sm.includes = append(sm.includes, g)
		}
	}

	return &sm, nil
}

// Match matches the files and invoke the call back function until there is an error.
func (s Matcher) Match(callback func(p string) error) error {
	return filepath.Walk(s.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rPath, err := filepath.Rel(s.dir, path)
		if err != nil {
			return err
		}

		for _, g := range s.excludes {
			matched, err := zglob.Match(g, rPath)
			if err != nil {
				return err
			} else if matched {
				return nil
			}
		}

		// if there is no includes, then all non-excludes files will be add to checksum
		if len(s.includes) == 0 {
			if err := callback(path); err != nil {
				return err
			}
		} else {
			for _, g := range s.includes {
				matched, err := zglob.Match(g, rPath)
				if err != nil {
					return err
				} else if matched {
					if err := callback(path); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func globs(dir string, globs []string) ([]string, error) {
	files := make([]string, 0)
	for _, g := range globs {
		f, err := glob(dir, g)
		if err != nil {
			continue
		}
		files = append(files, f...)
	}
	sort.Strings(files)
	return files, nil
}

func glob(dir string, g string) ([]string, error) {
	files := make([]string, 0)
	if !filepath.IsAbs(g) {
		g = filepath.Join(dir, g)
	}
	g, err := execext.Expand(g)
	if err != nil {
		return nil, err
	}
	fs, err := zglob.Glob(g)
	if err != nil {
		return nil, err
	}
	for _, f := range fs {
		info, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		files = append(files, f)
	}
	return files, nil
}
