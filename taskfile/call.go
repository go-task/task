package taskfile

// Call is the parameters to a task call
type Call struct {
	Dir string
	Task string
	Vars *Vars
}
