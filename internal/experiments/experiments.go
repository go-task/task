package experiments

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"golang.org/x/exp/slices"

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

func readDotEnv() {
	env, _ := godotenv.Read()
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
