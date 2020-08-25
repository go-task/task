package compiler

import (
	"os"
	"strings"

	"github.com/go-task/task/v3/taskfile"
)

// GetEnviron the all return all environment variables encapsulated on a
// taskfile.Vars
func GetEnviron() *taskfile.Vars {
	m := &taskfile.Vars{}
	for _, e := range os.Environ() {
		keyVal := strings.SplitN(e, "=", 2)
		key, val := keyVal[0], keyVal[1]
		m.Set(key, taskfile.Var{Static: val})
	}
	return m
}
