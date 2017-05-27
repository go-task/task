package task

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/go-task/task/execext"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig"
	"gopkg.in/yaml.v2"
)

var (
	// TaskvarsFilePath file containing additional variables
	TaskvarsFilePath = "Taskvars"
	// ErrMultilineResultCmd is returned when a command returns multiline result
	ErrMultilineResultCmd = errors.New("Got multiline result from command")
)

func handleDynamicVariableContent(value string) (string, error) {
	if !strings.HasPrefix(value, "$") {
		return value, nil
	}

	buff := bytes.NewBuffer(nil)

	opts := &execext.RunCommandOptions{
		Command: strings.TrimPrefix(value, "$"),
		Stdout:  buff,
		Stderr:  os.Stderr,
	}
	if err := execext.RunCommand(opts); err != nil {
		return "", err
	}

	result := buff.String()
	result = strings.TrimSuffix(result, "\n")
	if strings.ContainsRune(result, '\n') {
		return "", ErrMultilineResultCmd
	}

	result = strings.TrimSpace(result)
	return result, nil
}

func (t *Task) getVariables() (map[string]string, error) {
	localVariables := make(map[string]string)
	for key, value := range t.Vars {
		val, err := handleDynamicVariableContent(value)
		if err != nil {
			return nil, err
		}
		localVariables[key] = val
	}
	if fileVariables, err := readTaskvarsFile(); err == nil {
		for key, value := range fileVariables {
			val, err := handleDynamicVariableContent(value)
			if err != nil {
				return nil, err
			}
			localVariables[key] = val
		}
	} else {
		return nil, err
	}
	for key, value := range getEnvironmentVariables() {
		localVariables[key] = value
	}
	return localVariables, nil
}

var templateFuncs template.FuncMap

func init() {
	taskFuncs := template.FuncMap{
		"OS":   func() string { return runtime.GOOS },
		"ARCH": func() string { return runtime.GOARCH },
		// historical reasons
		"IsSH": func() bool { return true },
		"FromSlash": func(path string) string {
			return filepath.FromSlash(path)
		},
		"ToSlash": func(path string) string {
			return filepath.ToSlash(path)
		},
		"ExeExt": func() string {
			if runtime.GOOS == "windows" {
				return ".exe"
			}
			return ""
		},
	}

	templateFuncs = sprig.TxtFuncMap()
	for k, v := range taskFuncs {
		templateFuncs[k] = v
	}
}

// ReplaceSliceVariables writes vars into initial string slice
func (t *Task) ReplaceSliceVariables(initials []string) ([]string, error) {
	result := make([]string, len(initials))
	for i, s := range initials {
		var err error
		result[i], err = t.ReplaceVariables(s)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// ReplaceVariables writes vars into initial string
func (t *Task) ReplaceVariables(initial string) (string, error) {
	vars, err := t.getVariables()
	if err != nil {
		return "", err
	}

	templ, err := template.New("").Funcs(templateFuncs).Parse(initial)
	if err != nil {
		return "", err
	}

	b := bytes.NewBuffer(nil)
	if err = templ.Execute(b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}

// GetEnvironmentVariables returns environment variables as map
func getEnvironmentVariables() map[string]string {
	var (
		env = os.Environ()
		m   = make(map[string]string, len(env))
	)

	for _, e := range env {
		keyVal := strings.SplitN(e, "=", 2)
		key, val := keyVal[0], keyVal[1]
		m[key] = val
	}
	return m
}

func readTaskvarsFile() (map[string]string, error) {
	var variables map[string]string
	if b, err := ioutil.ReadFile(TaskvarsFilePath + ".yml"); err == nil {
		if err := yaml.Unmarshal(b, &variables); err != nil {
			return nil, err
		}
		return variables, nil
	}
	if b, err := ioutil.ReadFile(TaskvarsFilePath + ".json"); err == nil {
		if err := json.Unmarshal(b, &variables); err != nil {
			return nil, err
		}
		return variables, nil
	}
	if b, err := ioutil.ReadFile(TaskvarsFilePath + ".toml"); err == nil {
		if err := toml.Unmarshal(b, &variables); err != nil {
			return nil, err
		}
		return variables, nil
	}
	return variables, nil
}
