package summary

import (
	"strings"

	"github.com/go-task/task/v3/internal/log"
	"github.com/go-task/task/v3/taskfile"
)

func PrintTasks(t *taskfile.Taskfile, c []taskfile.Call) {
	for i, call := range c {
		PrintSpaceBetweenSummaries(i)
		PrintTask(t.Tasks.Get(call.Task))
	}
}

func PrintSpaceBetweenSummaries(i int) {
	spaceRequired := i > 0
	if !spaceRequired {
		return
	}

	log.Outf(log.Default, "\n")
	log.Outf(log.Default, "\n")
}

func PrintTask(t *taskfile.Task) {
	printTaskName(t)
	printTaskDescribingText(t)
	printTaskDependencies(t)
	printTaskAliases(t)
	printTaskCommands(t)
}

func printTaskDescribingText(t *taskfile.Task) {
	if hasSummary(t) {
		printTaskSummary(t)
	} else if hasDescription(t) {
		printTaskDescription(t)
	} else {
		printNoDescriptionOrSummary()
	}
}

func hasSummary(t *taskfile.Task) bool {
	return t.Summary != ""
}

func printTaskSummary(t *taskfile.Task) {
	lines := strings.Split(t.Summary, "\n")
	for i, line := range lines {
		notLastLine := i+1 < len(lines)
		if notLastLine || line != "" {
			log.Outf(log.Default, "%s\n", line)
		}
	}
}

func printTaskName(t *taskfile.Task) {
	log.Outf(log.Default, "task: ")
	log.Outf(log.Green, "%s\n", t.Name())
	log.Outf(log.Default, "\n")
}

func printTaskAliases(t *taskfile.Task) {
	if len(t.Aliases) == 0 {
		return
	}
	log.Outf(log.Default, "\n")
	log.Outf(log.Default, "aliases:\n")
	for _, alias := range t.Aliases {
		log.Outf(log.Default, " - ")
		log.Outf(log.Cyan, "%s\n", alias)
	}
}

func hasDescription(t *taskfile.Task) bool {
	return t.Desc != ""
}

func printTaskDescription(t *taskfile.Task) {
	log.Outf(log.Default, "%s\n", t.Desc)
}

func printNoDescriptionOrSummary() {
	log.Outf(log.Default, "(task does not have description or summary)\n")
}

func printTaskDependencies(t *taskfile.Task) {
	if len(t.Deps) == 0 {
		return
	}

	log.Outf(log.Default, "\n")
	log.Outf(log.Default, "dependencies:\n")

	for _, d := range t.Deps {
		log.Outf(log.Default, " - %s\n", d.Task)
	}
}

func printTaskCommands(t *taskfile.Task) {
	if len(t.Cmds) == 0 {
		return
	}

	log.Outf(log.Default, "\n")
	log.Outf(log.Default, "commands:\n")
	for _, c := range t.Cmds {
		isCommand := c.Cmd != ""
		log.Outf(log.Default, " - ")
		if isCommand {
			log.Outf(log.Yellow, "%s\n", c.Cmd)
		} else {
			log.Outf(log.Green, "Task: %s\n", c.Task)
		}
	}
}
