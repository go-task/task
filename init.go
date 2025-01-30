package task

import (
	"io"
	"os"

	"github.com/go-task/task/v3/errors"
)

const DefaultTaskfile = `# https://taskfile.dev

version: '3'

vars:
  GREETING: Hello, World!

tasks:
  default:
    cmds:
      - echo "{{.GREETING}}"
    silent: true
`

const DefaultTaskFilename = "Taskfile.yml"

// InitTaskfile creates a new Taskfile at path
func InitTaskfile(w io.Writer, path string) error {
	if _, err := os.Stat(path); err == nil {
		return errors.TaskfileAlreadyExistsError{}
	}

	if err := os.WriteFile(path, []byte(DefaultTaskfile), 0o644); err != nil {
		return err
	}

	return nil
}
