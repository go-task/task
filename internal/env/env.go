package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/taskfile/ast"
)

const taskVarPrefix = "TASK_"

// GetEnviron the all return all environment variables encapsulated on a
// ast.Vars
func GetEnviron() *ast.Vars {
	m := ast.NewVars()
	for _, e := range os.Environ() {
		keyVal := strings.SplitN(e, "=", 2)
		key, val := keyVal[0], keyVal[1]
		m.Set(key, ast.Var{Value: val})
	}
	return m
}

func Get(t *ast.Task) []string {
	if t.Env == nil {
		return nil
	}

	return GetFromVars(t.Env)
}

func GetFromVars(env *ast.Vars) []string {
	environ := os.Environ()

	for k, v := range env.ToCacheMap() {
		if !isTypeAllowed(v) {
			continue
		}
		if !experiments.EnvPrecedence.Enabled() {
			if _, alreadySet := os.LookupEnv(k); alreadySet {
				continue
			}
		}
		environ = append(environ, fmt.Sprintf("%s=%v", k, v))
	}

	return environ
}

func isTypeAllowed(v any) bool {
	switch v.(type) {
	case string, bool, int, float32, float64:
		return true
	default:
		return false
	}
}

func GetTaskEnv(key string) string {
	return os.Getenv(taskVarPrefix + key)
}
