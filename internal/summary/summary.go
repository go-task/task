package summary

import (
	"strings"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

func PrintTasks(l *logger.Logger, t *taskfile.Taskfile, c []taskfile.Call) {
	for i, call := range c {
		PrintSpaceBetweenSummaries(l, i)
		PrintTask(l, t.Tasks[call.Task])
	}
}

func PrintSpaceBetweenSummaries(l *logger.Logger, i int) {
	spaceRequired := i > 0
	if !spaceRequired {
		return
	}

	l.Outf(logger.Default, "")
	l.Outf(logger.Default, "")
}

func PrintTask(l *logger.Logger, t *taskfile.Task) {
	printTaskName(l, t)
	printTaskDescribingText(t, l)
	printTaskDependencies(l, t)
	printTaskCommands(l, t)
}

func printTaskDescribingText(t *taskfile.Task, l *logger.Logger) {
	if hasSummary(t) {
		printTaskSummary(l, t)
	} else if hasDescription(t) {
		printTaskDescription(l, t)
	} else {
		printNoDescriptionOrSummary(l)
	}
}

func hasSummary(t *taskfile.Task) bool {
	return t.Summary != ""
}

func printTaskSummary(l *logger.Logger, t *taskfile.Task) {
	lines := strings.Split(t.Summary, "\n")
	for i, line := range lines {
		notLastLine := i+1 < len(lines)
		if notLastLine || line != "" {
			l.Outf(logger.Default, line)
		}
	}
}

func printTaskName(l *logger.Logger, t *taskfile.Task) {
	l.Outf(logger.Default, "task: %s", t.Name())
	l.Outf(logger.Default, "")
}

func hasDescription(t *taskfile.Task) bool {
	return t.Desc != ""
}

func printTaskDescription(l *logger.Logger, t *taskfile.Task) {
	l.Outf(logger.Default, t.Desc)
}

func printNoDescriptionOrSummary(l *logger.Logger) {
	l.Outf(logger.Default, "(task does not have description or summary)")
}

func printTaskDependencies(l *logger.Logger, t *taskfile.Task) {
	if len(t.Deps) == 0 {
		return
	}

	l.Outf(logger.Default, "")
	l.Outf(logger.Default, "dependencies:")

	for _, d := range t.Deps {
		l.Outf(logger.Default, " - %s", d.Task)
	}
}

func printTaskCommands(l *logger.Logger, t *taskfile.Task) {
	if len(t.Cmds) == 0 {
		return
	}

	l.Outf(logger.Default, "")
	l.Outf(logger.Default, "commands:")
	for _, c := range t.Cmds {
		isCommand := c.Cmd != ""
		if isCommand {
			l.Outf(logger.Default, " - %s", c.Cmd)
		} else {
			l.Outf(logger.Default, " - Task: %s", c.Task)
		}
	}
}
