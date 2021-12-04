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
	// ErrIncludedTaskfilesCantHaveIncludes is returned when a included Taskfile contains includes
	ErrIncludedTaskfilesCantHaveIncludes = errors.New("task: Included Taskfiles can't have includes. Please, move the include to the main Taskfile")
	// ErrIncludedTaskfilesCantHaveDotenvs is returned when a included Taskfile contains dotenvs
	ErrIncludedTaskfilesCantHaveDotenvs = errors.New("task: Included Taskfiles can't have dotenv declarations. Please, move the dotenv declaration to the main Taskfile")

	defaultTaskfiles = []string{"Taskfile.yml", "Taskfile.yaml"}
)

// Taskfile reads a Taskfile for a given directory
// Uses current dir when dir is left empty. Uses Taskfile.yml
// or Taskfile.yaml when entrypoint is left empty
func Taskfile(dir string, entrypoint string) (*taskfile.Taskfile, error) {
	if dir == "" {
		d, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dir = d
	}
	path, err := exists(filepath.Join(dir, entrypoint))
	if err != nil {
		return nil, err
	}

	t, err := readTaskfile(path)
	if err != nil {
		return nil, err
	}

	v, err := t.ParsedVersion()
	if err != nil {
		return nil, err
	}

	err = t.Includes.Range(func(namespace string, includedTask taskfile.IncludedTaskfile) error {
		if v >= 3.0 {
			tr := templater.Templater{Vars: &taskfile.Vars{}, RemoveNoValue: true}
			includedTask = taskfile.IncludedTaskfile{
				Taskfile:       tr.Replace(includedTask.Taskfile),
				Dir:            tr.Replace(includedTask.Dir),
				Optional:       includedTask.Optional,
				AdvancedImport: includedTask.AdvancedImport,
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
			path = filepath.Join(dir, path)
		}

		path, err = exists(path)
		if err != nil {
			if includedTask.Optional {
				return nil
			}
			return err
		}

		includedTaskfile, err := readTaskfile(path)
		if err != nil {
			return err
		}
		if includedTaskfile.Includes.Len() > 0 {
			return ErrIncludedTaskfilesCantHaveIncludes
		}

		if v >= 3.0 && len(includedTaskfile.Dotenv) > 0 {
			return ErrIncludedTaskfilesCantHaveDotenvs
		}

		if includedTask.AdvancedImport {
			for k, v := range includedTaskfile.Vars.Mapping {
				o := v
				o.Dir = filepath.Join(dir, includedTask.Dir)
				includedTaskfile.Vars.Mapping[k] = o
			}
			for k, v := range includedTaskfile.Env.Mapping {
				o := v
				o.Dir = filepath.Join(dir, includedTask.Dir)
				includedTaskfile.Env.Mapping[k] = o
			}

			for _, task := range includedTaskfile.Tasks {
				if !filepath.IsAbs(task.Dir) {
					task.Dir = filepath.Join(includedTask.Dir, task.Dir)
				}
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
		path = filepath.Join(dir, fmt.Sprintf("Taskfile_%s.yml", runtime.GOOS))
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
