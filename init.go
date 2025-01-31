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

// InitTaskfile creates a new Taskfile at path.
//
// path can be either a file path or a directory path.
// If path is a directory, path/Taskfile.yml will be created.
func InitTaskfile(w io.Writer, path string) error {
	fi, err := os.Stat(path)
	if err == nil && !fi.IsDir() {
		return errors.TaskfileAlreadyExistsError{}
	}

	if fi != nil && fi.IsDir() {
		path = filepathext.SmartJoin(path, DefaultTaskFilename)
		// path was a directory, so check if Taskfile.yml exists in it
		if _, err := os.Stat(path); err == nil {
			return errors.TaskfileAlreadyExistsError{}
		}
	}

	if err := os.WriteFile(path, []byte(DefaultTaskfile), 0o644); err != nil {
		return err
	}

	return nil
}
