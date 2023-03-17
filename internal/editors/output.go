package editors

type (
	// Taskfile wraps task list output for use in editor integrations (e.g. VSCode, etc)
	Taskfile struct {
		Tasks    []Task `json:"tasks"`
		Location string `json:"location"`
	}
	// Task describes a single task
	Task struct {
		Name     string    `json:"name"`
		Desc     string    `json:"desc"`
		Summary  string    `json:"summary"`
		UpToDate bool      `json:"up_to_date"`
		Location *Location `json:"location"`
	}
	// Location describes a task's location in a taskfile
	Location struct {
		Line     int    `json:"line"`
		Column   int    `json:"column"`
		Taskfile string `json:"taskfile"`
	}
)
