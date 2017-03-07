package task

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"fmt"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

var (
	// TaskvarsFilePath file containing additional variables
	TaskvarsFilePath = "Taskvars"
	// DynamicVariablePattern is a pattern to test if a variable should get filled from running the content. It must contain a command group
	DynamicVariablePattern = "^@(?P<command>.*)" // alternative proposal: ^$((?P<command>.*))$
	// ErrCommandGroupNotFound returned when the command group is not present
	ErrCommandGroupNotFound = fmt.Errorf("%s does not contain the command group", DynamicVariablePattern)
)

func handleDynamicVariableContent(value string) (string, error) {
	if value == "" {
		return value, nil
	}
	re := regexp.MustCompile(DynamicVariablePattern)
	if !re.MatchString(value) {
		return value, nil
	}
	subExpressionIndex := 0
	for index, value := range re.SubexpNames() {
		if value == "command" {
			subExpressionIndex = index
			break
		}
	}
	if subExpressionIndex == 0 {
		return "", ErrCommandGroupNotFound
	}
	var cmd *exec.Cmd
	if ShExists {
		cmd = exec.Command(ShPath, "-c", re.FindStringSubmatch(value)[subExpressionIndex])
	} else {
		cmd = exec.Command("cmd", "/C", re.FindStringSubmatch(value)[subExpressionIndex])
	}
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	bytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

func (t Task) handleVariables() (map[string]string, error) {
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
