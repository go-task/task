package experiments

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
)

const envPrefix = "TASK_X_"

// A set of experiments that can be enabled or disabled.
var (
	GentleForce     Experiment
	RemoteTaskfiles Experiment
	AnyVariables    Experiment
	MapVariables    Experiment
	EnvPrecedence   Experiment
)

// An internal list of all the initialized experiments used for iterating.
var xList []Experiment

func init() {
	readDotEnv()
	GentleForce = New("GENTLE_FORCE", "1")
	RemoteTaskfiles = New("REMOTE_TASKFILES", "1")
	AnyVariables = New("ANY_VARIABLES")
	MapVariables = New("MAP_VARIABLES")
	EnvPrecedence = New("ENV_PRECEDENCE", "1")
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

func getEnvFilePath() string {
	// Parse the CLI flags again to get the directory/taskfile being run
	// We use a flagset here so that we can parse a subset of flags without exiting on error.
	var dir, taskfile string
	fs := pflag.NewFlagSet("experiments", pflag.ContinueOnError)
	fs.StringVarP(&dir, "dir", "d", "", "Sets directory of execution.")
	fs.StringVarP(&taskfile, "taskfile", "t", "", `Choose which Taskfile to run. Defaults to "Taskfile.yml".`)
	fs.Usage = func() {}
	_ = fs.Parse(os.Args[1:])
	// If the directory is set, find a .env file in that directory.
	if dir != "" {
		return filepath.Join(dir, ".env")
	}
	// If the taskfile is set, find a .env file in the directory containing the Taskfile.
	if taskfile != "" {
		return filepath.Join(filepath.Dir(taskfile), ".env")
	}
	// Otherwise just use the current working directory.
	return ".env"
}

func readDotEnv() {
	env, _ := godotenv.Read(getEnvFilePath())
	// If the env var is an experiment, set it.
	for key, value := range env {
		if strings.HasPrefix(key, envPrefix) {
			os.Setenv(key, value)
		}
	}
}
