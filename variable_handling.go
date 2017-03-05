package task

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

var (
	// VariableFilePath file containing additional variables
	VariableFilePath = "Taskvars"
)

func (t Task) handleVariables() (map[string]string, error) {
	localVariables := make(map[string]string)
	for key, value := range t.Variables {
		localVariables[key] = value
	}
	if fileVariables, err := readVariablefile(); err == nil {
		for key, value := range fileVariables {
			localVariables[key] = value
		}
	} else {
		return nil, err
	}
	for key, value := range getEnvironmentVariables() {
		localVariables[key] = value
	}
	return localVariables, nil
}

// ReplaceVariables writes variables into initial string
func ReplaceVariables(initial string, variables map[string]string) string {
	replaced := initial
	for name, val := range variables {
		replaced = strings.Replace(replaced, fmt.Sprintf("{{%s}}", name), val, -1)
	}
	return replaced
}

// GetEnvironmentVariables returns environment variables as map
func getEnvironmentVariables() map[string]string {
	getenvironment := func(data []string, getkeyval func(item string) (key, val string)) map[string]string {
		items := make(map[string]string)
		for _, item := range data {
			key, val := getkeyval(item)
			items[key] = val
		}
		return items
	}
	return getenvironment(os.Environ(), func(item string) (key, val string) {
		splits := strings.Split(item, "=")
		key = splits[0]
		val = splits[1]
		return
	})
}

func readVariablefile() (map[string]string, error) {
	var variables map[string]string
	if b, err := ioutil.ReadFile(VariableFilePath + ".yml"); err == nil {
		if err := yaml.Unmarshal(b, &variables); err != nil {
			return nil, err
		}
	}
	if b, err := ioutil.ReadFile(VariableFilePath + ".json"); err == nil {
		if err := json.Unmarshal(b, &variables); err != nil {
			return nil, err
		}
	}
	if b, err := ioutil.ReadFile(VariableFilePath + ".toml"); err == nil {
		if err := toml.Unmarshal(b, &variables); err != nil {
			return nil, err
		}
	}
	return variables, nil
}
