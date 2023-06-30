package experiments

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/internal/logger"
)

const envPrefix = "TASK_X_"

var GentleForce bool

func init() {
	readDotEnv()
	GentleForce = parseEnv("GENTLE_FORCE")
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

func List(l *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 6, ' ', 0)
	l.FOutf(w, logger.Yellow, "* ")
	l.FOutf(w, logger.Green, "GENTLE_FORCE")
	l.FOutf(w, logger.Default, ": \t%t\n", GentleForce)
	return w.Flush()
}
