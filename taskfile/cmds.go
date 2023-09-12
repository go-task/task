package taskfile

type Cmds struct {
	Pre           string
	Post          string
	TaskSeparator string `yaml:"task_separator"`
}
