package status

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/go-task/task/v2/internal/execext"

	"github.com/mattn/go-zglob"
)

func globs(dir string, globs []string) ([]string, error){
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
			continue
		}
		if info.IsDir() {
			continue
		}
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}
