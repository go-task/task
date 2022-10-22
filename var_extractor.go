package task

import (
	"github.com/go-task/task/v3/taskfile"
)

// extractAllVarsForTask extracts all vars and envs from the task, it's taskfile,
// and any dependencies, including taskvars.
// only the user-specified vars/envs will be extracted, i.e. not the whole OS environment.
func extractAllVarsForTask(
	call taskfile.Call,
	executor *Executor,
	task *taskfile.Task,
	taskEvaluatedVars *taskfile.Vars,
	additionalVars ...*taskfile.Vars,
) *map[string]string {
	// we want to export only the user-specified vars and envs
	excludeVars := []string{
		"CLI_ARGS",
	}

	result := taskfile.Vars{}

	// for some reason, task doesn't contain its vars, but it can be retrieved from the taskfile spec
	for _, taskfileTask := range executor.Taskfile.Tasks {
		if taskfileTask.Name() == task.Name() {
			if taskfileTask.Env != nil {
				result.Merge(taskfileTask.Env)
			}
			if taskfileTask.Vars != nil {
				result.Merge(taskfileTask.Vars)
			}
		}
	}

	if call.Vars != nil {
		result.Merge(call.Vars)
	}
	if task.Vars != nil {
		result.Merge(task.Vars)
	}
	if task.Env != nil {
		result.Merge(task.Env)
	}
	if task.IncludeVars != nil {
		result.Merge(task.IncludeVars)
	}
	if task.IncludedTaskfileVars != nil {
		result.Merge(task.IncludedTaskfileVars)
	}
	if len(additionalVars) > 0 {
		for _, additionalVar := range additionalVars {
			result.Merge(additionalVar)
		}
	}
	if executor.Taskfile.Vars != nil {
		result.Merge(executor.Taskfile.Vars)
	}

	// taskEvaluatedVars contains the whole env, we don't need it
	// but we need the evaluated dynamic vars values
	for varName := range result.Mapping {
		if evaluatedVar, ok := taskEvaluatedVars.Mapping[varName]; ok {
			result.Mapping[varName] = evaluatedVar
		}
	}

	for _, excludeVar := range excludeVars {
		delete(result.Mapping, excludeVar)
	}

	resultMapping := make(map[string]string, 0)

	for k, v := range result.Mapping {
		resultMapping[k] = v.Static
	}

	if result.Len() > 0 {
		return &resultMapping
	} else {
		return nil
	}
}
