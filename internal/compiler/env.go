package compiler

import (
	"os"
	"strings"

	"github.com/go-task/task/v2/internal/taskfile"
)

// GetEnviron the all return all environment variables encapsulated on a
// taskfile.Vars
func GetEnviron() taskfile.Vars {
	var (
		env = os.Environ()
		m   = make(taskfile.Vars, len(env))
	)

	for _, e := range env {
		keyVal := strings.SplitN(e, "=", 2)
		key, val := keyVal[0], keyVal[1]
		m[key] = taskfile.Var{Static: val}
	}
	return m
}
