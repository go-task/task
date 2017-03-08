package task

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

var (
	// TaskvarsFilePath file containing additional variables
	TaskvarsFilePath = "Taskvars"
)

func handleDynamicVariableContent(value string) (string, error) {
	if value == "" || value[0] != '$' {
		return value, nil
	}
	var cmd *exec.Cmd
	if ShExists {
		cmd = exec.Command(ShPath, "-c", value[1:])
	} else {
		cmd = exec.Command("cmd", "/C", value[1:])
	}
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	bytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

func (t *Task) handleVariables() (map[string]string, error) {
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
		val, err := handleDynamicVariableContent(value)
		if err != nil {
			return nil, err
		}
		localVariables[key] = val
	}
	return localVariables, nil
}

var templateFuncs = template.FuncMap{
	"OS":   func() string { return runtime.GOOS },
	"ARCH": func() string { return runtime.GOARCH },
	"IsSH": func() bool { return ShExists },
}

// ReplaceVariables writes vars into initial string
func ReplaceVariables(initial string, vars map[string]string) (string, error) {
	t, err := template.New("").Funcs(templateFuncs).Parse(initial)
	if err != nil {
		return "", err
	}
	b := bytes.NewBuffer(nil)
	if err = t.Execute(b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}

// GetEnvironmentVariables returns environment variables as map
func getEnvironmentVariables() map[string]string {
	type getKeyValFunc func(item string) (key, val string)
	getEnvironment := func(data []string, getKeyVal getKeyValFunc) map[string]string {
		items := make(map[string]string)
		for _, item := range data {
			key, val := getKeyVal(item)
			items[key] = val
		}
		return items
	}
	return getEnvironment(os.Environ(), func(item string) (key, val string) {
		splits := strings.Split(item, "=")
		key = splits[0]
		val = splits[1]
		return
	})
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
