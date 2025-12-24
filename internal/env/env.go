package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

// GetTaskEnvBool returns the boolean value of a TASK_ prefixed env var.
// Returns the value and true if set and valid, or false and false if not set or invalid.
func GetTaskEnvBool(key string) (bool, bool) {
	v := GetTaskEnv(key)
	if v == "" {
		return false, false
	}
	b, err := strconv.ParseBool(v)
	return b, err == nil
}

// GetTaskEnvInt returns the integer value of a TASK_ prefixed env var.
// Returns the value and true if set and valid, or 0 and false if not set or invalid.
func GetTaskEnvInt(key string) (int, bool) {
	v := GetTaskEnv(key)
	if v == "" {
		return 0, false
	}
	i, err := strconv.Atoi(v)
	return i, err == nil
}

// GetTaskEnvDuration returns the duration value of a TASK_ prefixed env var.
// Returns the value and true if set and valid, or 0 and false if not set or invalid.
func GetTaskEnvDuration(key string) (time.Duration, bool) {
	v := GetTaskEnv(key)
	if v == "" {
		return 0, false
	}
	d, err := time.ParseDuration(v)
	return d, err == nil
}

// GetTaskEnvString returns the string value of a TASK_ prefixed env var.
// Returns the value and true if set (non-empty), or empty string and false if not set.
func GetTaskEnvString(key string) (string, bool) {
	v := GetTaskEnv(key)
	return v, v != ""
}

// GetTaskEnvStringSlice returns a comma-separated list from a TASK_ prefixed env var.
// Returns the slice and true if set (non-empty), or nil and false if not set.
func GetTaskEnvStringSlice(key string) ([]string, bool) {
	v := GetTaskEnv(key)
	if v == "" {
		return nil, false
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}
