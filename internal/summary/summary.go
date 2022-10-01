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
	printTaskAliases(l, t)
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
	l.FOutf(l.Stdout, logger.Default, "task: ")
	l.FOutf(l.Stdout, logger.Green, "%s\n", t.Name())
	l.Outf(logger.Default, "")
}

func printTaskAliases(l *logger.Logger, t *taskfile.Task) {
	if len(t.Aliases) == 0 {
		return
	}
	l.Outf(logger.Default, "")
	l.Outf(logger.Default, "aliases:")
	for _, alias := range t.Aliases {
		l.FOutf(l.Stdout, logger.Default, " - ")
		l.Outf(logger.Cyan, alias)
	}
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
		l.FOutf(l.Stdout, logger.Default, " - ")
		if isCommand {
			l.FOutf(l.Stdout, logger.Yellow, "%s\n", c.Cmd)
		} else {
			l.FOutf(l.Stdout, logger.Green, "Task: %s\n", c.Task)
		}
	}
}
