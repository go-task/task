package read

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

var (
	// ErrIncludedTaskfilesCantHaveIncludes is returned when a included Taskfile contains includes
	ErrIncludedTaskfilesCantHaveIncludes = errors.New("task: Included Taskfiles can't have includes. Please, move the include to the main Taskfile")
	// ErrIncludedTaskfilesCantHaveDotenvs is returned when a included Taskfile contains dotenvs
	ErrIncludedTaskfilesCantHaveDotenvs = errors.New("task: Included Taskfiles can't have dotenv declarations. Please, move the dotenv declaration to the main Taskfile")
)

// Taskfile reads a Taskfile for a given directory
func Taskfile(dir string, entrypoint string) (*taskfile.Taskfile, error) {
	path := filepath.Join(dir, entrypoint)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf(`task: No Taskfile found on "%s". Use "task --init" to create a new one`, path)
	}
	t, err := readTaskfile(path)
	if err != nil {
		return nil, err
	}

	v, err := t.ParsedVersion()
	if err != nil {
		return nil, err
	}

	if v >= 3.0 && len(t.Dotenv) > 0 {
		for _, dotEnvPath := range t.Dotenv {
			if !filepath.IsAbs(dotEnvPath) {
				dotEnvPath = filepath.Join(dir, dotEnvPath)
			}
			// allow for missing env files since they may be created by a bootstrap task
			if _, err := os.Stat(dotEnvPath); !os.IsNotExist(err) {
				envs, err := godotenv.Read(dotEnvPath)
				if err != nil {
					return nil, err
				}
				for key, value := range envs {
					if _, ok := t.Env.Mapping[key]; !ok {
						t.Env.Set(key, taskfile.Var{Static: value})
					}
				}
			} else {
			}
		}
	}

	for namespace, includedTask := range t.Includes {
		if v >= 3.0 {
			tr := templater.Templater{Vars: &taskfile.Vars{}, RemoveNoValue: true}
			includedTask = taskfile.IncludedTaskfile{
				Taskfile:       tr.Replace(includedTask.Taskfile),
				Dir:            tr.Replace(includedTask.Dir),
				AdvancedImport: includedTask.AdvancedImport,
			}
			if err := tr.Err(); err != nil {
				return nil, err
			}
		}

		if filepath.IsAbs(includedTask.Taskfile) {
			path = includedTask.Taskfile
		} else {
			path = filepath.Join(dir, includedTask.Taskfile)
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			path = filepath.Join(path, "Taskfile.yml")
		}
		includedTaskfile, err := readTaskfile(path)
		if err != nil {
			return nil, err
		}
		if len(includedTaskfile.Includes) > 0 {
			return nil, ErrIncludedTaskfilesCantHaveIncludes
		}

		if v >= 3.0 && len(includedTaskfile.Dotenv) > 0 {
			return nil, ErrIncludedTaskfilesCantHaveDotenvs
		}

		if includedTask.AdvancedImport {
			for _, task := range includedTaskfile.Tasks {
				if !filepath.IsAbs(task.Dir) {
					task.Dir = filepath.Join(includedTask.Dir, task.Dir)
				}
			}
		}

		if err = taskfile.Merge(t, includedTaskfile, namespace); err != nil {
			return nil, err
		}
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
