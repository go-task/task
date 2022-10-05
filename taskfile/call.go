package taskfile

// Call is the parameters to a task call
type Call struct {
	Task string `json:"task"`
	Vars *Vars  `json:"vars"`
}
