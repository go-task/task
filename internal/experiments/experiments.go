package experiments

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"gopkg.in/yaml.v3"

	"github.com/Ladicle/tabwriter"
	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
)

const (
	envPrefix             = "TASK_X_"
	defaultConfigFilename = ".task-experiments.yml"
)

// A set of experiments that can be enabled or disabled.
type ExperimentConfigFile struct {
	Experiments map[string]string `yaml:",inline"`
}
var (
	GentleForce     Experiment
	RemoteTaskfiles Experiment
	AnyVariables    Experiment
	MapVariables    Experiment
	EnvPrecedence   Experiment
)

// An internal list of all the initialized experiments used for iterating.
var xList []Experiment

var ExperimentConfig ExperimentConfigFile

func init() {
	readDotEnv()
	ExperimentConfig = readConfig()
	GentleForce = New("GENTLE_FORCE", "1")
	RemoteTaskfiles = New("REMOTE_TASKFILES", "1")
	AnyVariables = New("ANY_VARIABLES")
	MapVariables = New("MAP_VARIABLES", "1", "2")
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

func getFilePath(filename string) string {
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
		return filepath.Join(dir, filename)
	}
	// If the taskfile is set, find a .env file in the directory containing the Taskfile.
	if taskfile != "" {
		return filepath.Join(filepath.Dir(taskfile), filename)
	}
	// Otherwise just use the current working directory.
	return filename
}

func readDotEnv() {
	env, _ := godotenv.Read(getFilePath(".env"))
	// If the env var is an experiment, set it.
	for key, value := range env {
		if strings.HasPrefix(key, envPrefix) {
			os.Setenv(key, value)
		}
	}
}

func readConfig() ExperimentConfigFile {
	var cfg ExperimentConfigFile
	filename := getFilePath(defaultConfigFilename)
	}
	if err != nil {
	content, err := os.ReadFile(filename)
		return ExperimentConfigFile{}

	if err := yaml.Unmarshal(content, &cfg); err != nil {
	}
		return ExperimentConfigFile{}

}
	return cfg