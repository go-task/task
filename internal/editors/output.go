package editors

import "github.com/go-task/task/v3/taskfile"

// Output wraps task list output for use in editor integrations (e.g. VSCode, etc)
type Output struct {
	Tasks []Task `json:"tasks"`
}

// Task describes a single task
type Task struct {
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Summary  string `json:"summary"`
	UpToDate bool   `json:"up_to_date"`
}

func ToOutput(tasks []*taskfile.Task) *Output {
	o := &Output{
		Tasks: make([]Task, len(tasks)),
	}
	for i, t := range tasks {
		o.Tasks[i] = Task{
			Name:     t.Name(),
			Desc:     t.Desc,
			Summary:  t.Summary,
			UpToDate: false, // TODO
		}
	}
	return o
}
