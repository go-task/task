package task

import (
	"io/ioutil"
	"log"
	"os"
)

const defaultTaskfile = `# github.com/go-task/task

default:
  cmds:
    - echo "Hello, World!"
`

// InitTaskfile Taskfile creates a new Taskfile
func InitTaskfile() error {
	for _, f := range []string{"Taskfile.yml", "Taskfile.toml", "Taskfile.json"} {
		if _, err := os.Stat(f); err == nil {
			return ErrTaskfileAlreadyExists
		}
	}

	if err := ioutil.WriteFile("Taskfile.yml", []byte(defaultTaskfile), 0666); err != nil {
		return err
	}
	log.Printf("Taskfile.yml created in the current directory")
	return nil
}
