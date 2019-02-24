package summary

import (
	"github.com/go-task/task/v2/internal/logger"
	"github.com/go-task/task/v2/internal/taskfile"
	"strings"
)

func Print(Logger *logger.Logger, task *taskfile.Task) {
	printTaskName(Logger, task)
	printTaskSummary(task.Summary, Logger)
	printTaskDependencies(task.Deps, Logger)
	printTaskCommands(task.Cmds, Logger)
}

func printTaskName(Logger *logger.Logger, task *taskfile.Task) {
	Logger.Outf("task: " + task.Task)
	Logger.Outf("")
}

func printTaskCommands(cmds []*taskfile.Cmd, logger *logger.Logger) {
	hasCommands := len(cmds) > 0
	if hasCommands {
		logger.Outf("")
		logger.Outf("commands:")
		for _, c := range cmds {
			isCommand := c.Cmd != ""
			if isCommand {
				logger.Outf(" - %s", c.Cmd)
			} else {
				logger.Outf(" - Task: %s", c.Task)
			}
		}
	}
}

func printTaskDependencies(deps []*taskfile.Dep, logger *logger.Logger) {
	hasDependencies := len(deps) > 0
	if hasDependencies {
		logger.Outf("")
		logger.Outf("dependencies:")

		for _, d := range deps {
			logger.Outf(" - %s", d.Task)
		}
	}
}

func printTaskSummary(description string, Logger *logger.Logger) {
	lines := strings.Split(description, "\n")
	for i, line := range lines {
		notLastLine := i+1 < len(lines)
		if notLastLine || line != "" {
			Logger.Outf(line)
		}
	}
}
