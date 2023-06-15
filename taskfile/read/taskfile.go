package read

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/sysinfo"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

var (
	// ErrIncludedTaskfilesCantHaveDotenvs is returned when a included Taskfile contains dotenvs
	ErrIncludedTaskfilesCantHaveDotenvs = errors.New("task: Included Taskfiles can't have dotenv declarations. Please, move the dotenv declaration to the main Taskfile")

	defaultTaskfiles = []string{
		"Taskfile.yml",
		"taskfile.yml",
		"Taskfile.yaml",
		"taskfile.yaml",
		"Taskfile.dist.yml",
		"taskfile.dist.yml",
		"Taskfile.dist.yaml",
		"taskfile.dist.yaml",
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
func Taskfile(readerNode *ReaderNode) (*taskfile.Taskfile, string, error) {
	if readerNode.Dir == "" {
		d, err := os.Getwd()
		if err != nil {
			return nil, "", err
		}
		readerNode.Dir = d
	}

	path, err := existsWalk(filepathext.SmartJoin(readerNode.Dir, readerNode.Entrypoint))
	if err != nil {
		return nil, "", err
	}
	readerNode.Dir = filepath.Dir(path)
	readerNode.Entrypoint = filepath.Base(path)

	t, err := readTaskfile(path)
	if err != nil {
		return nil, "", err
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
		if t.Version.Compare(taskfile.V3) >= 0 {
			tr := templater.Templater{Vars: t.Vars, RemoveNoValue: true}
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

		includedTaskfile, _, err := Taskfile(includeReaderNode)
		if err != nil {
			if includedTask.Optional {
				return nil
			}
			return err
		}

		if t.Version.Compare(taskfile.V3) >= 0 && len(includedTaskfile.Dotenv) > 0 {
			return ErrIncludedTaskfilesCantHaveDotenvs
		}

		if includedTask.AdvancedImport {
			dir, err := includedTask.FullDirPath()
			if err != nil {
				return err
			}

			// nolint: errcheck
			includedTaskfile.Vars.Range(func(k string, v taskfile.Var) error {
				o := v
				o.Dir = dir
				includedTaskfile.Vars.Set(k, o)
				return nil
			})
			// nolint: errcheck
			includedTaskfile.Env.Range(func(k string, v taskfile.Var) error {
				o := v
				o.Dir = dir
				includedTaskfile.Env.Set(k, o)
				return nil
			})

			for _, task := range includedTaskfile.Tasks.Values() {
				task.Dir = filepathext.SmartJoin(dir, task.Dir)
				task.IncludeVars = includedTask.Vars
				task.IncludedTaskfileVars = includedTaskfile.Vars
				task.IncludedTaskfile = &includedTask
			}
		}

		if err = taskfile.Merge(t, includedTaskfile, &includedTask, namespace); err != nil {
			return err
		}

		if includedTaskfile.Tasks.Get("default") != nil && t.Tasks.Get(namespace) == nil {
			defaultTaskName := fmt.Sprintf("%s:default", namespace)
			task := t.Tasks.Get(defaultTaskName)
			task.Aliases = append(task.Aliases, namespace)
			task.Aliases = append(task.Aliases, includedTask.Aliases...)
			t.Tasks.Set(defaultTaskName, task)
		}

		return nil
	})
	if err != nil {
		return nil, "", err
	}

	if t.Version.Compare(taskfile.V3) < 0 {
		path = filepathext.SmartJoin(readerNode.Dir, fmt.Sprintf("Taskfile_%s.yml", runtime.GOOS))
		if _, err = os.Stat(path); err == nil {
			osTaskfile, err := readTaskfile(path)
			if err != nil {
				return nil, "", err
			}
			if err = taskfile.Merge(t, osTaskfile, nil); err != nil {
				return nil, "", err
			}
		}
	}

	// Set the location of the Taskfile
	t.Location = path

	for _, task := range t.Tasks.Values() {
		// If the task is not defined, create a new one
		if task == nil {
			task = &taskfile.Task{}
		}
		// Set the location of the taskfile for each task
		if task.Location.Taskfile == "" {
			task.Location.Taskfile = path
		}
	}

	return t, readerNode.Dir, nil
}

func readTaskfile(file string) (*taskfile.Taskfile, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var t taskfile.Taskfile
	if err := yaml.NewDecoder(f).Decode(&t); err != nil {
		return nil, &errors.TaskfileInvalidError{FilePath: filepathext.TryAbsToRel(file), Err: err}
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

	return "", errors.TaskfileNotFoundError{Dir: path, Walk: false}
}

func existsWalk(path string) (string, error) {
	origPath := path
	owner, err := sysinfo.Owner(path)
	if err != nil {
		return "", err
	}
	for {
		fpath, err := exists(path)
		if err == nil {
			return fpath, nil
		}

		// Get the parent path/user id
		parentPath := filepath.Dir(path)
		parentOwner, err := sysinfo.Owner(parentPath)
		if err != nil {
			return "", err
		}

		// Error if we reached the root directory and still haven't found a file
		// OR if the user id of the directory changes
		if path == parentPath || (parentOwner != owner) {
			return "", errors.TaskfileNotFoundError{Dir: origPath, Walk: false}
		}

		owner = parentOwner
		path = parentPath
	}
}

func checkCircularIncludes(node *ReaderNode) error {
	if node == nil {
		return errors.New("task: failed to check for include cycle: node was nil")
	}
	if node.Parent == nil {
		return errors.New("task: failed to check for include cycle: node.Parent was nil")
	}
	curNode := node
	basePath := filepathext.SmartJoin(node.Dir, node.Entrypoint)
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
