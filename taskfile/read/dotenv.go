package read

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/internal/compiler"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

func Dotenv(c compiler.Compiler, tf *taskfile.Taskfile, dir string) (*taskfile.Vars, error) {
	if len(tf.Dotenv) == 0 {
		return nil, nil
	}

	vars, err := c.GetTaskfileVariables()
	if err != nil {
		return nil, err
	}

	env := &taskfile.Vars{}

	tr := templater.Templater{Vars: vars, RemoveNoValue: true}

	for _, dotEnvPath := range tf.Dotenv {
		dotEnvPath = tr.Replace(dotEnvPath)

		if !filepath.IsAbs(dotEnvPath) {
			dotEnvPath = filepath.Join(dir, dotEnvPath)
		}
		if _, err := os.Stat(dotEnvPath); os.IsNotExist(err) {
			continue
		}

		envs, err := godotenv.Read(dotEnvPath)
		if err != nil {
			return nil, err
		}
		for key, value := range envs {
			if _, ok := env.Mapping[key]; !ok {
				env.Set(key, taskfile.Var{Static: value})
			}
		}
	}

	return env, nil
}
