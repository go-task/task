package read

import (
	"fmt"
	"os"
	"runtime"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile"
)

// Taskvars reads a Taskvars for a given directory
func Taskvars(dir string) (*taskfile.Vars, error) {
	vars := &taskfile.Vars{}

	path := filepathext.SmartJoin(dir, "Taskvars.yml")
	if _, err := os.Stat(path); err == nil {
		vars, err = readTaskvars(path)
		if err != nil {
			return nil, err
		}
	}

	path = filepathext.SmartJoin(dir, fmt.Sprintf("Taskvars_%s.yml", runtime.GOOS))
	if _, err := os.Stat(path); err == nil {
		osVars, err := readTaskvars(path)
		if err != nil {
			return nil, err
		}
		vars.Merge(osVars)
	}

	return vars, nil
}

func readTaskvars(file string) (*taskfile.Vars, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	var vars taskfile.Vars
	return &vars, yaml.NewDecoder(f).Decode(&vars)
}
