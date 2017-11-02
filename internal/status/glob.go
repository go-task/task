package status

import (
	"path/filepath"
	"sort"

	"github.com/mattn/go-zglob"
	"github.com/mitchellh/go-homedir"
)

func glob(dir string, globs []string) (files []string, err error) {
	for _, g := range globs {
		if !filepath.IsAbs(g) {
			g = filepath.Join(dir, g)
		}
		g, err = homedir.Expand(g)
		if err != nil {
			return nil, err
		}
		f, err := zglob.Glob(g)
		if err != nil {
			return nil, err
		}
		files = append(files, f...)
	}
	sort.Strings(files)
	return
}
