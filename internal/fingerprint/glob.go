package fingerprint

import (
	"os"
	"sort"

	"github.com/mattn/go-zglob"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

func globs(dir string, globs []string) ([]string, error) {
	files := make([]string, 0)
	for _, g := range globs {
		f, err := Glob(dir, g)
		if err != nil {
			continue
		}
		files = append(files, f...)
	}
	sort.Strings(files)
	return files, nil
}

func Glob(dir string, g string) ([]string, error) {
	files := make([]string, 0)
	g = filepathext.SmartJoin(dir, g)

	g, err := execext.Expand(g)
	if err != nil {
		return nil, err
	}

	fs, err := zglob.GlobFollowSymlinks(g)
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
