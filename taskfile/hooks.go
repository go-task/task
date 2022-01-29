package taskfile

// Hooks represents events fired during a task's execution. Hooks do not stop
// task execution if any of their commands fail.
type Hooks struct {
	// BeforeAll commands are called before a task starts execution
	BeforeAll []*Cmd `yaml:"before_all"`
	// AfterAll commands are called after a task completes, success or failure
	AfterAll []*Cmd `yaml:"after_all"`
	// OnSuccess commands are called when a task completes successfully
	OnSuccess []*Cmd `yaml:"on_success"`
	// OnFailure commands are called when a task fails with an error
	OnFailure []*Cmd `yaml:"on_failure"`
	// OnSkipped commands are called when a task is skipped due to status, precondition or checksum
	OnSkipped []*Cmd `yaml:"on_skipped"`
	/**
	// Other useful hooks?

	// OnAbort commands are called when a task is aborted with Ctrl+C
	OnAbort   []*Cmd `yaml:"on_abort"`
	// OnRetry commands are called when a task retries after an error
	OnRetry   []*Cmd `yaml:"on_retry"`
	// OnTimeout commands are called when a task hits a timeout error
	OnTimeout []*Cmd `yaml:"on_timeout"`
	**/
}
