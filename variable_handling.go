package task

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/go-task/task/execext"

	"github.com/Masterminds/sprig"
)

var (
	// TaskvarsFilePath file containing additional variables
	TaskvarsFilePath = "Taskvars"
	// ErrMultilineResultCmd is returned when a command returns multiline result
	ErrMultilineResultCmd = errors.New("Got multiline result from command")
)

func (e *Executor) handleDynamicVariableContent(value string) (string, error) {
	if !strings.HasPrefix(value, "$") {
		return value, nil
	}

	e.muDynamicCache.Lock()
	defer e.muDynamicCache.Unlock()
	if result, ok := e.dynamicCache[value]; ok {
		return result, nil
	}

	buff := bytes.NewBuffer(nil)

	opts := &execext.RunCommandOptions{
		Command: strings.TrimPrefix(value, "$"),
		Dir:     e.Dir,
		Stdout:  buff,
		Stderr:  e.Stderr,
	}
	if err := execext.RunCommand(opts); err != nil {
		return "", &dynamicVarError{cause: err, cmd: opts.Command}
	}

	result := buff.String()
	result = strings.TrimSuffix(result, "\n")
	if strings.ContainsRune(result, '\n') {
		return "", ErrMultilineResultCmd
	}

	result = strings.TrimSpace(result)
	e.verbosePrintfln(`task: dynamic variable: "%s", result: "%s"`, value, result)
	e.dynamicCache[value] = result
	return result, nil
}

func (e *Executor) getVariables(call Call) (Vars, error) {
	t := e.Tasks[call.Task]

	result := make(Vars, len(t.Vars)+len(e.taskvars)+len(call.Vars))
	merge := func(vars Vars) error {
		for k, v := range vars {
			v, err := e.handleDynamicVariableContent(v)
			if err != nil {
				return err
			}
			result[k] = v
		}
		return nil
	}

	if err := merge(e.taskvars); err != nil {
		return nil, err
	}
	if err := merge(t.Vars); err != nil {
		return nil, err
	}
	if err := merge(getEnvironmentVariables()); err != nil {
		return nil, err
	}
	if err := merge(call.Vars); err != nil {
		return nil, err
	}

	return result, nil
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
func (e *Executor) ReplaceSliceVariables(initials []string, call Call) ([]string, error) {
	result := make([]string, len(initials))
	for i, s := range initials {
		var err error
		result[i], err = e.ReplaceVariables(s, call)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// ReplaceVariables writes vars into initial string
func (e *Executor) ReplaceVariables(initial string, call Call) (string, error) {
	vars, err := e.getVariables(call)
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
func getEnvironmentVariables() Vars {
	var (
		env = os.Environ()
		m   = make(Vars, len(env))
	)

	for _, e := range env {
		keyVal := strings.SplitN(e, "=", 2)
		key, val := keyVal[0], keyVal[1]
		m[key] = val
	}
	return m
}
