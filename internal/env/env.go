package env

import (
	"fmt"
	"os"

	"github.com/go-task/task/v3/taskfile"
)

func Get(t *taskfile.Task) []string {
	if t.Env == nil {
		return nil
	}

	environ := os.Environ()

	for k, v := range t.Env.ToCacheMap() {
		str, isString := v.(string)
		if !isString {
			continue
		}

		if _, alreadySet := os.LookupEnv(k); alreadySet {
			continue
		}

		environ = append(environ, fmt.Sprintf("%s=%s", k, str))
	}

	return environ
}
