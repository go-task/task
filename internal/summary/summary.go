package summary

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

func PrintTasks(l *logger.Logger, t *ast.Taskfile, c []string) {
	for i, call := range c {
		PrintSpaceBetweenSummaries(l, i)
		if task, ok := t.Tasks.Get(call); ok {
			PrintTask(l, task)
		}
	}
}

func PrintSpaceBetweenSummaries(l *logger.Logger, i int) {
	spaceRequired := i > 0
	if !spaceRequired {
		return
	}

	l.Outf(logger.Default, "\n")
	l.Outf(logger.Default, "\n")
}

func PrintTask(l *logger.Logger, t *ast.Task) {
	printTaskName(l, t)
	printTaskDescribingText(t, l)
	printTaskVars(l, t)
	printTaskEnv(l, t)
	printTaskRequires(l, t)
	printTaskDependencies(l, t)
	printTaskAliases(l, t)
	printTaskCommands(l, t)
}

func printTaskDescribingText(t *ast.Task, l *logger.Logger) {
	if hasSummary(t) {
		printTaskSummary(l, t)
	} else if hasDescription(t) {
		printTaskDescription(l, t)
	} else {
		printNoDescriptionOrSummary(l)
	}
}

func hasSummary(t *ast.Task) bool {
	return t.Summary != ""
}

func printTaskSummary(l *logger.Logger, t *ast.Task) {
	lines := strings.Split(t.Summary, "\n")
	for i, line := range lines {
		notLastLine := i+1 < len(lines)
		if notLastLine || line != "" {
			l.Outf(logger.Default, "%s\n", line)
		}
	}
}

func printTaskName(l *logger.Logger, t *ast.Task) {
	l.Outf(logger.Default, "task: ")
	l.Outf(logger.Green, "%s\n", t.Name())
	l.Outf(logger.Default, "\n")
}

func printTaskAliases(l *logger.Logger, t *ast.Task) {
	if len(t.Aliases) == 0 {
		return
	}
	l.Outf(logger.Default, "\n")
	l.Outf(logger.Default, "aliases:\n")
	for _, alias := range t.Aliases {
		l.Outf(logger.Default, " - ")
		l.Outf(logger.Cyan, "%s\n", alias)
	}
}

func hasDescription(t *ast.Task) bool {
	return t.Desc != ""
}

func printTaskDescription(l *logger.Logger, t *ast.Task) {
	l.Outf(logger.Default, "%s\n", t.Desc)
}

func printNoDescriptionOrSummary(l *logger.Logger) {
	l.Outf(logger.Default, "(task does not have description or summary)\n")
}

func printTaskDependencies(l *logger.Logger, t *ast.Task) {
	if len(t.Deps) == 0 {
		return
	}

	l.Outf(logger.Default, "\n")
	l.Outf(logger.Default, "dependencies:\n")

	for _, d := range t.Deps {
		l.Outf(logger.Default, " - %s\n", d.Task)
	}
}

func printTaskCommands(l *logger.Logger, t *ast.Task) {
	if len(t.Cmds) == 0 {
		return
	}

	l.Outf(logger.Default, "\n")
	l.Outf(logger.Default, "commands:\n")
	for _, c := range t.Cmds {
		isCommand := c.Cmd != ""
		l.Outf(logger.Default, " - ")
		if isCommand {
			l.Outf(logger.Yellow, "%s\n", c.Cmd)
		} else {
			l.Outf(logger.Green, "Task: %s\n", c.Task)
		}
	}
}

func printTaskVars(l *logger.Logger, t *ast.Task) {
	if t.Vars == nil || t.Vars.Len() == 0 {
		return
	}

	osEnvVars := getEnvVarNames()

	taskfileEnvVars := make(map[string]bool)
	if t.Env != nil {
		for key := range t.Env.All() {
			taskfileEnvVars[key] = true
		}
	}

	hasNonEnvVars := false
	for key := range t.Vars.All() {
		if !isEnvVar(key, osEnvVars) && !taskfileEnvVars[key] {
			hasNonEnvVars = true
			break
		}
	}

	if !hasNonEnvVars {
		return
	}

	l.Outf(logger.Default, "\n")
	l.Outf(logger.Default, "vars:\n")

	for key, value := range t.Vars.All() {
		// Only display variables that are not from OS environment or Taskfile env
		if !isEnvVar(key, osEnvVars) && !taskfileEnvVars[key] {
			formattedValue := formatVarValue(value)
			l.Outf(logger.Yellow, "  %s: %s\n", key, formattedValue)
		}
	}
}

func printTaskEnv(l *logger.Logger, t *ast.Task) {
	if t.Env == nil || t.Env.Len() == 0 {
		return
	}

	envVars := getEnvVarNames()

	hasNonEnvVars := false
	for key := range t.Env.All() {
		if !isEnvVar(key, envVars) {
			hasNonEnvVars = true
			break
		}
	}

	if !hasNonEnvVars {
		return
	}

	l.Outf(logger.Default, "\n")
	l.Outf(logger.Default, "env:\n")

	for key, value := range t.Env.All() {
		// Only display variables that are not from OS environment
		if !isEnvVar(key, envVars) {
			formattedValue := formatVarValue(value)
			l.Outf(logger.Yellow, "  %s: %s\n", key, formattedValue)
		}
	}
}

// formatVarValue formats a variable value based on its type.
// Handles static values, shell commands (sh:), references (ref:), and maps.
func formatVarValue(v ast.Var) string {
	// Shell command - check this first before Value
	// because dynamic vars may have both Sh and an empty Value
	if v.Sh != nil {
		return fmt.Sprintf("sh: %s", *v.Sh)
	}

	// Reference
	if v.Ref != "" {
		return fmt.Sprintf("ref: %s", v.Ref)
	}

	// Static value
	if v.Value != nil {
		// Check if it's a map or complex type
		if m, ok := v.Value.(map[string]any); ok {
			return formatMap(m, 4)
		}
		// Simple string value
		return fmt.Sprintf(`"%v"`, v.Value)
	}

	return `""`
}

// formatMap formats a map value with proper indentation for YAML.
func formatMap(m map[string]any, indent int) string {
	if len(m) == 0 {
		return "{}"
	}

	var result strings.Builder
	result.WriteString("\n")
	spaces := strings.Repeat(" ", indent)

	for k, v := range m {
		result.WriteString(fmt.Sprintf("%s%s: %v\n", spaces, k, v))
	}

	return result.String()
}

func printTaskRequires(l *logger.Logger, t *ast.Task) {
	if t.Requires == nil || len(t.Requires.Vars) == 0 {
		return
	}

	l.Outf(logger.Default, "\n")
	l.Outf(logger.Default, "requires:\n")
	l.Outf(logger.Default, "  vars:\n")

	for _, v := range t.Requires.Vars {
		// If the variable has enum constraints, format accordingly
		if len(v.Enum) > 0 {
			l.Outf(logger.Yellow, "    - %s:\n", v.Name)
			l.Outf(logger.Yellow, "        enum:\n")
			for _, enumValue := range v.Enum {
				l.Outf(logger.Yellow, "          - %s\n", enumValue)
			}
		} else {
			// Simple required variable
			l.Outf(logger.Yellow, "    - %s\n", v.Name)
		}
	}
}

func getEnvVarNames() map[string]bool {
	envMap := make(map[string]bool)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) > 0 {
			envMap[parts[0]] = true
		}
	}
	return envMap
}

// isEnvVar checks if a variable is from OS environment or auto-generated by Task.
func isEnvVar(key string, envVars map[string]bool) bool {
	// Filter out auto-generated Task variables
	if strings.HasPrefix(key, "TASK_") ||
		strings.HasPrefix(key, "CLI_") ||
		strings.HasPrefix(key, "ROOT_") ||
		key == "TASK" ||
		key == "TASKFILE" ||
		key == "TASKFILE_DIR" ||
		key == "USER_WORKING_DIR" ||
		key == "ALIAS" ||
		key == "MATCH" {
		return true
	}
	return envVars[key]
}
