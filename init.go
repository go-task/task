package task

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const defaultTaskfile = `# https://taskfile.dev

version: 2

vars:
  GREETING: Hello, World!

tasks:
  default:
    cmds:
      - echo "{{.GREETING}}"
    silent: true
`

// InitTaskfile Taskfile creates a new Taskfile
func InitTaskfile(w io.Writer, dir string) error {
	f := filepath.Join(dir, "Taskfile.yml")

	if _, err := os.Stat(f); err == nil {
		return ErrTaskfileAlreadyExists
	}

	if err := ioutil.WriteFile(f, []byte(defaultTaskfile), 0644); err != nil {
		return err
	}
	fmt.Fprintf(w, "Taskfile.yml created in the current directory\n")
	return nil
}
