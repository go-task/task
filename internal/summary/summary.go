package summary

import (
	"github.com/go-task/task/v2/internal/logger"
	"github.com/go-task/task/v2/internal/taskfile"
	"strings"
)

func Print(l *logger.Logger, t *taskfile.Task) {
	printTaskName(l, t)
	if hasSummary(t) {
		printTaskSummary(l, t)
	} else {
		printTaskDescription(l, t)
	}
	printTaskDependencies(l, t)
	printTaskCommands(l, t)
}

func hasSummary(task *taskfile.Task) bool {
	return task.Summary != ""
}

func printTaskName(Logger *logger.Logger, task *taskfile.Task) {
	Logger.Outf("task: " + task.Task)
	Logger.Outf("")
}

func printTaskCommands(logger *logger.Logger, task *taskfile.Task) {
	hasCommands := len(task.Cmds) > 0
	if hasCommands {
		logger.Outf("")
		logger.Outf("commands:")
		for _, c := range task.Cmds {
			isCommand := c.Cmd != ""
			if isCommand {
				logger.Outf(" - %s", c.Cmd)
			} else {
				logger.Outf(" - Task: %s", c.Task)
			}
		}
	}
}

func printTaskDependencies(logger *logger.Logger, task *taskfile.Task) {
	hasDependencies := len(task.Deps) > 0
	if hasDependencies {
		logger.Outf("")
		logger.Outf("dependencies:")

		for _, d := range task.Deps {
			logger.Outf(" - %s", d.Task)
		}
	}
}

func printTaskSummary(Logger *logger.Logger, task *taskfile.Task) {
	lines := strings.Split(task.Summary, "\n")
	for i, line := range lines {
		notLastLine := i+1 < len(lines)
		if notLastLine || line != "" {
			Logger.Outf(line)
		}
	}
}

func printTaskDescription(Logger *logger.Logger, task *taskfile.Task) {
	Logger.Outf(task.Desc)
}
