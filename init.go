package task

import (
	"io"
	"os"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
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

// InitTaskfile creates a new Taskfile
func InitTaskfile(w io.Writer, dir string) error {
	f := filepathext.SmartJoin(dir, DefaultTaskFilename)

	if _, err := os.Stat(f); err == nil {
		return errors.TaskfileAlreadyExistsError{}
	}

	if err := os.WriteFile(f, []byte(DefaultTaskfile), 0o644); err != nil {
		return err
	}

	return nil
}
