package task

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const defaultTaskfile = `# github.com/go-task/task

default:
  cmds:
    - echo "Hello, World!"
`

// InitTaskfile Taskfile creates a new Taskfile
func InitTaskfile(path string) error {
	for _, f := range []string{"Taskfile.yml", "Taskfile.toml", "Taskfile.json"} {
		f = filepath.Join(path, f)
		if _, err := os.Stat(f); err == nil {
			return ErrTaskfileAlreadyExists
		}
	}

	f := filepath.Join(path, "Taskfile.yml")
	if err := ioutil.WriteFile(f, []byte(defaultTaskfile), 0666); err != nil {
		return err
	}
	log.Printf("Taskfile.yml created in the current directory")
	return nil
}
