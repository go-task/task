package taskfile

// Inputs represents a group of Input
type Inputs map[string]*Input

// Input represents an interactive input variable
type Input struct {
	Desc      string
	Required  bool
	Default   string
	Validator string
}
