package summary

import (
	"strings"

	"github.com/go-task/task/v2/internal/logger"
	"github.com/go-task/task/v2/internal/taskfile"
)

func Print(l *logger.Logger, t *taskfile.Task) {
	printTaskName(l, t)
	if hasSummary(t) {
		printTaskSummary(l, t)
	} else if hasDescription(t) {
		printTaskDescription(l, t)
	} else {
		printNoDescriptionOrSummary(l)
	}
	printTaskDependencies(l, t)
	printTaskCommands(l, t)
}

func hasSummary(t *taskfile.Task) bool {
	return t.Summary != ""
}

func printTaskSummary(l *logger.Logger, t *taskfile.Task) {
	lines := strings.Split(t.Summary, "\n")
	for i, line := range lines {
		notLastLine := i+1 < len(lines)
		if notLastLine || line != "" {
			l.Outf(line)
		}
	}
}

func printTaskName(l *logger.Logger, t *taskfile.Task) {
	l.Outf("task: %s", t.Task)
	l.Outf("")
}

func hasDescription(t *taskfile.Task) bool {
	return t.Desc != ""
}

func printTaskDescription(l *logger.Logger, t *taskfile.Task) {
	l.Outf(t.Desc)
}

func printNoDescriptionOrSummary(l *logger.Logger) {
	l.Outf("(task does not have description or summary)")
}

func printTaskDependencies(l *logger.Logger, t *taskfile.Task) {
	hasDependencies := len(t.Deps) > 0
	if hasDependencies {
		l.Outf("")
		l.Outf("dependencies:")

		for _, d := range t.Deps {
			l.Outf(" - %s", d.Task)
		}
	}
}

func printTaskCommands(l *logger.Logger, t *taskfile.Task) {
	hasCommands := len(t.Cmds) > 0
	if hasCommands {
		l.Outf("")
		l.Outf("commands:")
		for _, c := range t.Cmds {
			isCommand := c.Cmd != ""
			if isCommand {
				l.Outf(" - %s", c.Cmd)
			} else {
				l.Outf(" - Task: %s", c.Task)
			}
		}
	}
}
