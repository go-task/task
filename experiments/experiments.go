package experiments

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/taskrc"
	"github.com/go-task/task/v3/taskrc/ast"
)

const envPrefix = "TASK_X_"

// Active experiments.
var (
	GentleForce     Experiment
	RemoteTaskfiles Experiment
	EnvPrecedence   Experiment
)

// Inactive experiments. These are experiments that cannot be enabled, but are
// preserved for error handling.
var (
	AnyVariables Experiment
	MapVariables Experiment
)

// An internal list of all the initialized experiments used for iterating.
var xList []Experiment

func Parse(dir string) {
	config, _ := taskrc.GetConfig(dir)
	ParseWithConfig(dir, config)
}

func ParseWithConfig(dir string, config *ast.TaskRC) {
	// Read any .env files
	readDotEnv(dir)
	// Initialize the experiments
	GentleForce = New("GENTLE_FORCE", config, 1)
	RemoteTaskfiles = New("REMOTE_TASKFILES", config, 1)
	EnvPrecedence = New("ENV_PRECEDENCE", config, 1)
	AnyVariables = New("ANY_VARIABLES", config)
	MapVariables = New("MAP_VARIABLES", config)
}

// Validate checks if any experiments have been enabled while being inactive.
// If one is found, the function returns an error.
func Validate() error {
	for _, x := range List() {
		if err := x.Valid(); err != nil {
			return err
		}
	}
	return nil
}

func List() []Experiment {
	return xList
}

func getEnv(xName string) string {
	envName := fmt.Sprintf("%s%s", envPrefix, xName)
	return os.Getenv(envName)
}

func getFilePath(filename, dir string) string {
	if dir != "" {
		return filepath.Join(dir, filename)
	}
	return filename
}

func readDotEnv(dir string) {
	env, err := godotenv.Read(getFilePath(".env", dir))
	if err != nil {
		return
	}

	// If the env var is an experiment, set it.
	for key, value := range env {
		if strings.HasPrefix(key, envPrefix) {
			os.Setenv(key, value)
		}
	}
}
