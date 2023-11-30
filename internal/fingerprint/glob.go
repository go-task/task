package fingerprint

import (
	"os"
	"sort"

	"github.com/mattn/go-zglob"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile"
)

func Globs(dir string, globs []*taskfile.Glob) ([]string, error) {
	fileMap := make(map[string]bool)
	for _, g := range globs {
		matches, err := Glob(dir, g.Glob)
		if err != nil {
			continue
		}
		for _, match := range matches {
			fileMap[match] = !g.Negate
		}
	}
	files := make([]string, 0)
	for file, includePath := range fileMap {
		if includePath {
			files = append(files, file)
		}
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
