package editors

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
