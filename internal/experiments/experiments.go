package experiments

import (
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"

	"github.com/go-task/task/v3/internal/logger"
)

const envPrefix = "TASK_X_"

type Experiment struct {
	Name    string
	Enabled bool
	Value   string
}

// A list of experiments.
var (
	GentleForce     Experiment
	RemoteTaskfiles Experiment
	AnyVariables    Experiment
)

func init() {
	readDotEnv()
	GentleForce = New("GENTLE_FORCE")
	RemoteTaskfiles = New("REMOTE_TASKFILES")
	AnyVariables = New("ANY_VARIABLES", "1", "2")
}

func New(xName string, enabledValues ...string) Experiment {
	if len(enabledValues) == 0 {
		enabledValues = []string{"1"}
	}
	value := getEnv(xName)
	return Experiment{
		Name:    xName,
		Enabled: slices.Contains(enabledValues, value),
		Value:   value,
	}
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
	_ = fs.Parse(os.Args[1:])
	// If the directory is set, find a .env file in that directory.
	if dir != "" {
		return path.Join(dir, ".env")
	}
	// If the taskfile is set, find a .env file in the directory containing the Taskfile.
	if taskfile != "" {
		return path.Join(path.Dir(taskfile), ".env")
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

func printExperiment(w io.Writer, l *logger.Logger, x Experiment) {
	l.FOutf(w, logger.Yellow, "* ")
	l.FOutf(w, logger.Green, x.Name)
	l.FOutf(w, logger.Default, ": \t%s\n", x.String())
}

func List(l *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, ' ', 0)
	printExperiment(w, l, GentleForce)
	printExperiment(w, l, RemoteTaskfiles)
	printExperiment(w, l, AnyVariables)
	return w.Flush()
}
