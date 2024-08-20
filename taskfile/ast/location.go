package ast

type Location struct {
	Line        int
	Column      int
	Taskfile    string
	TaskfileDir string
}

func (l *Location) DeepCopy() *Location {
	if l == nil {
		return nil
	}
	return &Location{
		Line:        l.Line,
		Column:      l.Column,
		Taskfile:    l.Taskfile,
		TaskfileDir: l.TaskfileDir,
	}
}
