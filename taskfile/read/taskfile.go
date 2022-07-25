package read

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

var (
	// ErrIncludedTaskfilesCantHaveDotenvs is returned when a included Taskfile contains dotenvs
	ErrIncludedTaskfilesCantHaveDotenvs = errors.New("task: Included Taskfiles can't have dotenv declarations. Please, move the dotenv declaration to the main Taskfile")

	defaultTaskfiles = []string{
		"Taskfile.yml",
		"Taskfile.yaml",
		"Taskfile.dist.yml",
		"Taskfile.dist.yaml",
	}
)

type ReaderNode struct {
	Dir        string
	Entrypoint string
	Optional   bool
	Parent     *ReaderNode
}

// Taskfile reads a Taskfile for a given directory
// Uses current dir when dir is left empty. Uses Taskfile.yml
// or Taskfile.yaml when entrypoint is left empty
func Taskfile(readerNode *ReaderNode) (*taskfile.Taskfile, error) {
	if readerNode.Dir == "" {
		d, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		readerNode.Dir = d
	}
	path, err := exists(filepath.Join(readerNode.Dir, readerNode.Entrypoint))
	if err != nil {
		return nil, err
	}
	readerNode.Entrypoint = filepath.Base(path)

	t, err := readTaskfile(path)
	if err != nil {
		return nil, err
	}

	v, err := t.ParsedVersion()
	if err != nil {
		return nil, err
	}

	// Resolve any relative paths used by includes to absolute ones
	_ = t.Includes.Range(func(key string, includedFile taskfile.IncludedTaskfile) error {
		if !filepath.IsAbs(includedFile.Taskfile) {
			includedFile.Taskfile = filepath.Join(readerNode.Dir, includedFile.Taskfile)
		}

		if includedFile.Dir != "" && !filepath.IsAbs(includedFile.Dir) {
			includedFile.Dir = filepath.Join(readerNode.Dir, includedFile.Dir)
		}

		t.Includes.Set(key, includedFile)
		return nil
	})

	err = t.Includes.Range(func(namespace string, includedTask taskfile.IncludedTaskfile) error {
		if v >= 3.0 {
			tr := templater.Templater{Vars: &taskfile.Vars{}, RemoveNoValue: true}
			includedTask = taskfile.IncludedTaskfile{
				Taskfile:       tr.Replace(includedTask.Taskfile),
				Dir:            tr.Replace(includedTask.Dir),
				Optional:       includedTask.Optional,
				AdvancedImport: includedTask.AdvancedImport,
				Vars:           includedTask.Vars,
			}
			if err := tr.Err(); err != nil {
				return err
			}
		}

		path, err := execext.Expand(includedTask.Taskfile)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(readerNode.Dir, path)
		}
		path, err = exists(path)
		if err != nil {
			if includedTask.Optional {
				return nil
			}
			return err
		}

		includeReaderNode := &ReaderNode{
			Dir:        filepath.Dir(path),
			Entrypoint: filepath.Base(path),
			Parent:     readerNode,
			Optional:   includedTask.Optional,
		}

		if err := checkCircularIncludes(includeReaderNode); err != nil {
			return err
		}

		includedTaskfile, err := Taskfile(includeReaderNode)
		if err != nil {
			if includedTask.Optional {
				return nil
			}
			return err
		}

		if v >= 3.0 && len(includedTaskfile.Dotenv) > 0 {
			return ErrIncludedTaskfilesCantHaveDotenvs
		}

		if includedTask.AdvancedImport {
			for k, v := range includedTaskfile.Vars.Mapping {
				o := v
				o.Dir = filepath.Join(readerNode.Dir, includedTask.Dir)
				includedTaskfile.Vars.Mapping[k] = o
			}
			for k, v := range includedTaskfile.Env.Mapping {
				o := v
				o.Dir = filepath.Join(readerNode.Dir, includedTask.Dir)
				includedTaskfile.Env.Mapping[k] = o
			}

			for _, task := range includedTaskfile.Tasks {
				if !filepath.IsAbs(task.Dir) {
					task.Dir = filepath.Join(includedTask.Dir, task.Dir)
				}
				task.IncludeVars = includedTask.Vars
				task.IncludedTaskfileVars = includedTaskfile.Vars
			}
		}

		if err = taskfile.Merge(t, includedTaskfile, namespace); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if v < 3.0 {
		path = filepath.Join(readerNode.Dir, fmt.Sprintf("Taskfile_%s.yml", runtime.GOOS))
		if _, err = os.Stat(path); err == nil {
			osTaskfile, err := readTaskfile(path)
			if err != nil {
				return nil, err
			}
			if err = taskfile.Merge(t, osTaskfile); err != nil {
				return nil, err
			}
		}
	}

	for name, task := range t.Tasks {
		if task == nil {
			task = &taskfile.Task{}
			t.Tasks[name] = task
		}
		task.Task = name
	}

	return t, nil
}

func readTaskfile(file string) (*taskfile.Taskfile, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	var t taskfile.Taskfile
	return &t, yaml.NewDecoder(f).Decode(&t)
}

func exists(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if fi.Mode().IsRegular() {
		return path, nil
	}

	for _, n := range defaultTaskfiles {
		fpath := filepath.Join(path, n)
		if _, err := os.Stat(fpath); err == nil {
			return fpath, nil
		}
	}

	return "", fmt.Errorf(`task: No Taskfile found in "%s". Use "task --init" to create a new one`, path)
}

func checkCircularIncludes(node *ReaderNode) error {
	if node == nil {
		return errors.New("task: failed to check for include cycle: node was nil")
	}
	if node.Parent == nil {
		return errors.New("task: failed to check for include cycle: node.Parent was nil")
	}
	var curNode = node
	var basePath = filepath.Join(node.Dir, node.Entrypoint)
	for curNode.Parent != nil {
		curNode = curNode.Parent
		curPath := filepath.Join(curNode.Dir, curNode.Entrypoint)
		if curPath == basePath {
			return fmt.Errorf("task: include cycle detected between %s <--> %s",
				curPath,
				filepath.Join(node.Parent.Dir, node.Parent.Entrypoint),
			)
		}
	}
	return nil
}
