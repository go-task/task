package experiments

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const envPrefix = "TASK_X_"

var defaultConfigFilenames = []string{
	".taskrc.yml",
	".taskrc.yaml",
}

type experimentConfigFile struct {
	Experiments map[string]int `yaml:"experiments"`
	Version     *semver.Version
}

// Active experiments.
var (
	GentleForce     = New("GENTLE_FORCE", 1)
	RemoteTaskfiles = New("REMOTE_TASKFILES", 1)
	EnvPrecedence   = New("ENV_PRECEDENCE", 1)
)

// Inactive experiments. These are experiments that cannot be enabled, but are
// preserved for error handling.
var (
	AnyVariables = New("ANY_VARIABLES")
	MapVariables = New("MAP_VARIABLES")
)

// An internal list of all the initialized experiments used for iterating.
var (
	xList            []Experiment
	experimentConfig experimentConfigFile
)

func Parse(dir string) {
	readDotEnv(dir)
	experimentConfig = readConfig(dir)
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
	env, _ := godotenv.Read(getFilePath(".env", dir))
	// If the env var is an experiment, set it.
	for key, value := range env {
		if strings.HasPrefix(key, envPrefix) {
			os.Setenv(key, value)
		}
	}
}

func readConfig(dir string) experimentConfigFile {
	var cfg experimentConfigFile

	var content []byte
	var err error
	for _, filename := range defaultConfigFilenames {
		path := getFilePath(filename, dir)
		content, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return experimentConfigFile{}
	}

	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return experimentConfigFile{}
	}

	return cfg
}
