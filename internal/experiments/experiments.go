package experiments

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
)

const envPrefix = "TASK_X_"

type Experiment struct {
	Name    string // The name of the experiment.
	Active  bool   // Whether it is possible to enable this experiment.
	Enabled bool   // Whether this experiment is enabled.
	Value   string // The version of the experiment that is enabled.
}

// An internal list of all the initialized experiments used for iterating.
var xList []Experiment

// A set of experiments that can be enabled or disabled.
var (
	GentleForce     Experiment
	RemoteTaskfiles Experiment
	AnyVariables    Experiment
	MapVariables    Experiment
	EnvPrecedence   Experiment
)

func init() {
	readDotEnv()
	GentleForce = newExperiment("GENTLE_FORCE", "1")
	RemoteTaskfiles = newExperiment("REMOTE_TASKFILES", "1")
	AnyVariables = newExperiment("ANY_VARIABLES")
	MapVariables = newExperiment("MAP_VARIABLES", "1", "2")
	EnvPrecedence = newExperiment("ENV_PRECEDENCE", "1")
}

func newExperiment(xName string, enabledValues ...string) Experiment {
	value := getEnv(xName)
	x := Experiment{
		Name:    xName,
		Active:  len(enabledValues) > 0,
		Enabled: slices.Contains(enabledValues, value),
		Value:   value,
	}
	xList = append(xList, x)
	return x
}

func (x Experiment) String() string {
	if x.Enabled {
		return fmt.Sprintf("on (%s)", x.Value)
	}
	return "off"
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

// Validate checks if any experiments have been enabled while being inactive.
// If one is found, the function returns an error.
func Validate() error {
	for _, x := range List() {
		if !x.Active && x.Value != "" {
			return fmt.Errorf("task: Experiment %q is not active and cannot be enabled", x.Name)
		}
	}
	return nil
}

func List() []Experiment {
	return xList
}
