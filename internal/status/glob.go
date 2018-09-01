package status

import (
	"path/filepath"
	"sort"

	"github.com/mattn/go-zglob"
	"mvdan.cc/sh/shell"
)

func glob(dir string, globs []string) (files []string, err error) {
	for _, g := range globs {
		if !filepath.IsAbs(g) {
			g = filepath.Join(dir, g)
		}
		g, err = shell.Expand(g, nil)
		if err != nil {
			return nil, err
		}
		f, err := zglob.Glob(g)
		if err != nil {
			continue
		}
		files = append(files, f...)
	}
	sort.Strings(files)
	return
}
