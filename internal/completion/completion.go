package completion

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

type TemplateValues struct {
	Entrypoint string
	Flags      []*pflag.Flag
	Tasks      []*taskfile.Task
}

//go:embed templates/*
var templates embed.FS

func Compile(completion string, tasks taskfile.Tasks) (string, error) {
	// Get the file extension for the selected shell
	var ext string
	switch completion {
	case "bash":
		ext = "bash"
	case "fish":
		ext = "fish"
	case "powershell":
		ext = "ps1"
	case "zsh":
		ext = "zsh"
	default:
		return "", fmt.Errorf("unknown completion shell: %s", completion)
	}

	// Load the template
	templateName := fmt.Sprintf("task.tpl.%s", ext)
	tpl, err := template.New(templateName).
		Funcs(templater.TemplateFuncs).
		ParseFS(templates, filepath.Join("templates", templateName))
	if err != nil {
		return "", err
	}

	values := TemplateValues{
		Entrypoint: os.Args[0],
		Flags:      getFlagNames(),
		Tasks:      tasks.Values(),
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, values); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getFlagNames() []*pflag.Flag {
	var flags []*pflag.Flag
	pflag.VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, flag)
	})
	return flags
}
