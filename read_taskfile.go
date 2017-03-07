package task

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

func readTaskfile() (map[string]*Task, error) {
	initialTasks, err := readTaskfileData(TaskFilePath)
	if err != nil {
		return nil, err
	}
	mergeTasks, err := readTaskfileData(fmt.Sprintf("%s_%s", TaskFilePath, runtime.GOOS))
	if err != nil {
		switch err.(type) {
		default:
			return nil, err
		case taskFileNotFound:
			return initialTasks, nil
		}
	}
	if err := mergo.MapWithOverwrite(&initialTasks, mergeTasks); err != nil {
		return nil, err
	}
	return initialTasks, nil
}

func readTaskfileData(path string) (tasks map[string]*Task, err error) {
	if b, err := ioutil.ReadFile(path + ".yml"); err == nil {
		return tasks, yaml.Unmarshal(b, &tasks)
	}
	if b, err := ioutil.ReadFile(path + ".json"); err == nil {
		return tasks, json.Unmarshal(b, &tasks)
	}
	if b, err := ioutil.ReadFile(path + ".toml"); err == nil {
		return tasks, toml.Unmarshal(b, &tasks)
	}
	return nil, taskFileNotFound{path}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
