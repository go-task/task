package task

import (
	"fmt"
	"os"
	"strings"
)

func (t Task) handleVariables() map[string]string {
	localVariables := make(map[string]string)
	for key, value := range t.Variables {
		localVariables[key] = value
	}
	for key, value := range getEnvironmentVariables() {
		localVariables[key] = value
	}
	return localVariables
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
