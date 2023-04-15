package task

import (
	"fmt"
	"io"
	"os"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
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

const defaultTaskfileName = "Taskfile.yml"

// InitTaskfile Taskfile creates a new Taskfile
func InitTaskfile(w io.Writer, dir string) error {
	f := filepathext.SmartJoin(dir, defaultTaskfileName)

	if _, err := os.Stat(f); err == nil {
		return errors.TaskfileAlreadyExistsError{}
	}

	if err := os.WriteFile(f, []byte(defaultTaskfile), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s created in the current directory\n", defaultTaskfile)
	return nil
}
