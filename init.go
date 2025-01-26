package task

import (
	"fmt"
	"io"
	"os"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
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

// InitTaskfile creates a new Taskfile
//
// verbosity specifies how much to print to the terminal:
//
// 0 = Don't print anything
//
// 1 = Print filename only
//
// 2 = Print file contents + filename
func InitTaskfile(w io.Writer, dir string, verbosity uint8, l *logger.Logger) error {
	f := filepathext.SmartJoin(dir, defaultTaskfileName)

	if _, err := os.Stat(f); err == nil {
		return errors.TaskfileAlreadyExistsError{}
	}

	if err := os.WriteFile(f, []byte(defaultTaskfile), 0o644); err != nil {
		return err
	}

	if verbosity > 0 {
		if verbosity > 1 {
			fmt.Fprintf(w, "%s\n", defaultTaskfile)
		}
		l.Outf(logger.Green, "%s created in the current directory\n", defaultTaskfileName)
	}
	return nil
}
