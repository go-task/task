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

func initTaskfile() {
	for _, f := range []string{"Taskfile.yml", "Taskfile.toml", "Taskfile.json"} {
		if _, err := os.Stat(f); err == nil {
			log.Printf("A Taskfile already exists")
			os.Exit(1)
			return
		}
	}

	if err := ioutil.WriteFile("Taskfile.yml", []byte(defaultTaskfile), 0666); err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("Taskfile.yml created in the current directory")
}
