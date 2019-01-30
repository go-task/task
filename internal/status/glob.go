package status

import (
	"path/filepath"
	"sort"

	"github.com/go-task/task/v2/internal/execext"

	"github.com/mattn/go-zglob"
)

func glob(dir string, globs []string) (files []string, err error) {
	for _, g := range globs {
		if !filepath.IsAbs(g) {
			g = filepath.Join(dir, g)
		}
		g, err = execext.Expand(filepath.ToSlash(g))
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
