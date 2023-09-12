package experiments

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/internal/logger"
)

const envPrefix = "TASK_X_"

// A list of experiments.
var (
	GentleForce     bool
	RemoteTaskfiles bool
)

func init() {
	readDotEnv()
	GentleForce = parseEnv("GENTLE_FORCE")
	RemoteTaskfiles = parseEnv("REMOTE_TASKFILES")
}

func parseEnv(xName string) bool {
	envName := fmt.Sprintf("%s%s", envPrefix, xName)
	return os.Getenv(envName) == "1"
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

func printExperiment(w io.Writer, l *logger.Logger, name string, value bool) {
	l.FOutf(w, logger.Yellow, "* ")
	l.FOutf(w, logger.Green, name)
	l.FOutf(w, logger.Default, ": \t%t\n", value)
}

func List(l *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, ' ', 0)
	printExperiment(w, l, "GENTLE_FORCE", GentleForce)
	printExperiment(w, l, "REMOTE_TASKFILES", RemoteTaskfiles)
	return w.Flush()
}
