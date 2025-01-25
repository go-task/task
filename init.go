package task

import (
	"fmt"
	"io"
	"os"

	"github.com/go-task/task/v3/errors"
)

const defaultTaskfile = `# https://taskfile.dev

version: '3'

vars:
  GREETING: Hello, World!

tasks:
  default:
    cmds:
      - echo "{{.GREETING}}"
    silent: true
`

// InitTaskfile Taskfile creates a new Taskfile
func InitTaskfile(w io.Writer, filepath string) error {
	if _, err := os.Stat(filepath); err == nil {
		return errors.TaskfileAlreadyExistsError{}
	}

	if err := os.WriteFile(filepath, []byte(defaultTaskfile), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s created in the current directory\n", defaultTaskfile)
	return nil
}
