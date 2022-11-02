package read

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/filepathext"
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

	path, err := exists(filepathext.SmartJoin(readerNode.Dir, readerNode.Entrypoint))
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

	// Annotate any included Taskfile reference with a base directory for resolving relative paths
	_ = t.Includes.Range(func(key string, includedFile taskfile.IncludedTaskfile) error {
		// Set the base directory for resolving relative paths, but only if not already set
		if includedFile.BaseDir == "" {
			includedFile.BaseDir = readerNode.Dir
			t.Includes.Set(key, includedFile)
		}
		return nil
	})

	err = t.Includes.Range(func(namespace string, includedTask taskfile.IncludedTaskfile) error {
		if v >= 3.0 {
			tr := templater.Templater{Vars: &taskfile.Vars{}, RemoveNoValue: true}
			includedTask = taskfile.IncludedTaskfile{
				Taskfile:       tr.Replace(includedTask.Taskfile),
				Dir:            tr.Replace(includedTask.Dir),
				Optional:       includedTask.Optional,
				Internal:       includedTask.Internal,
				Aliases:        includedTask.Aliases,
				AdvancedImport: includedTask.AdvancedImport,
				Vars:           includedTask.Vars,
				BaseDir:        includedTask.BaseDir,
			}
			if err := tr.Err(); err != nil {
				return err
			}
		}

		path, err := includedTask.FullTaskfilePath()
		if err != nil {
			return err
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
			dir, err := includedTask.FullDirPath()
			if err != nil {
				return err
			}

			for k, v := range includedTaskfile.Vars.Mapping {
				o := v
				o.Dir = dir
				includedTaskfile.Vars.Mapping[k] = o
			}
			for k, v := range includedTaskfile.Env.Mapping {
				o := v
				o.Dir = dir
				includedTaskfile.Env.Mapping[k] = o
			}

			for _, task := range includedTaskfile.Tasks {
				task.Dir = filepathext.SmartJoin(dir, task.Dir)
				task.IncludeVars = includedTask.Vars
				task.IncludedTaskfileVars = includedTaskfile.Vars
				task.IncludedTaskfile = &includedTask
			}
		}

		if err = taskfile.Merge(t, includedTaskfile, &includedTask, namespace); err != nil {
			return err
		}

		if includedTaskfile.Tasks["default"] != nil && t.Tasks[namespace] == nil {
			defaultTaskName := fmt.Sprintf("%s:default", namespace)
			t.Tasks[defaultTaskName].Aliases = append(t.Tasks[defaultTaskName].Aliases, namespace)
			t.Tasks[defaultTaskName].Aliases = append(t.Tasks[defaultTaskName].Aliases, includedTask.Aliases...)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if v < 3.0 {
		path = filepathext.SmartJoin(readerNode.Dir, fmt.Sprintf("Taskfile_%s.yml", runtime.GOOS))
		if _, err = os.Stat(path); err == nil {
			osTaskfile, err := readTaskfile(path)
			if err != nil {
				return nil, err
			}
			if err = taskfile.Merge(t, osTaskfile, nil); err != nil {
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
	if err := yaml.NewDecoder(f).Decode(&t); err != nil {
		return nil, fmt.Errorf("task: Failed to parse %s:\n%w", filepathext.TryAbsToRel(file), err)
	}
	return &t, nil
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
		fpath := filepathext.SmartJoin(path, n)
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
	var basePath = filepathext.SmartJoin(node.Dir, node.Entrypoint)
	for curNode.Parent != nil {
		curNode = curNode.Parent
		curPath := filepathext.SmartJoin(curNode.Dir, curNode.Entrypoint)
		if curPath == basePath {
			return fmt.Errorf("task: include cycle detected between %s <--> %s",
				curPath,
				filepathext.SmartJoin(node.Parent.Dir, node.Parent.Entrypoint),
			)
		}
	}
	return nil
}
