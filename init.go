package task

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultTaskfileHeader = `# https://taskfile.dev

`

type Tasks map[string]*Task

type Task struct {
	Task   string   `yaml:"task,omitempty"`
	Cmds   []string `yaml:"cmds,omitempty"`
	Silent bool     `yaml:"silent,omitempty"`
}

type Vars map[string]string

type Taskfile struct {
	Version string `yaml:"version,omitempty"`
	Vars    *Vars  `yaml:"vars,omitempty"`
	Tasks   Tasks  `yaml:"tasks,omitempty"`
	Silent  bool   `yaml:"silent,omitempty"`
}

// InitTaskfile Taskfile creates a new Taskfile
func InitTaskfile(w io.Writer, dir string) error {
	f := filepath.Join(dir, "Taskfile.yaml")

	if _, err := os.Stat(f); err == nil {
		return ErrTaskfileAlreadyExists
	}

	taskfile := Taskfile{
		Version: "3",
		Vars: &Vars{
			"GREETING": "Hello, World!",
		},
		Tasks: Tasks{
			"default": &Task{
				Cmds: []string{
					"echo \"{{.GREETING}}\"",
				},
				Silent: true,
			},
		},
	}
	out, err := yaml.Marshal(taskfile)
	if err != nil {
		return err
	}
	data := []byte(defaultTaskfileHeader)
	data = append(data, out...)
	if err := os.WriteFile(f, data, 0644); err != nil {
		return err
	}
	fmt.Fprintf(w, "Taskfile.yaml created in the current directory\n")
	return nil
}
