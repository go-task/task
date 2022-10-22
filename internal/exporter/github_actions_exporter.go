package exporter

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/google/uuid"
	"os"
	"regexp"
	"runtime"
)

type uuidGenerator func() uuid.UUID

const gitHubEnvName = "GITHUB_ENV"

// GithubActionsExporter exports the vars to GITHUB_ENV
type GithubActionsExporter struct {
	envFilePath   string
	uuidGenerator uuidGenerator
	eol           string
}

func NewGithubActionsExporter() (*GithubActionsExporter, error) {
	githubEnvFilePath, envExists := os.LookupEnv(gitHubEnvName)
	if !envExists {
		return nil, errors.New("GITHUB_ENV variable is not set")
	}

	// https://stackoverflow.com/a/49963413/7519767
	eol := "\n"
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}

	return &GithubActionsExporter{
		envFilePath:   githubEnvFilePath,
		uuidGenerator: uuid.New,
		eol:           eol,
	}, nil
}

// Export sets the vars evaluated values to the GitHub Actions environment
func (e *GithubActionsExporter) Export(vars map[string]string) error {
	f, err := os.OpenFile(e.envFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0600)
	if err != nil {
		return fmt.Errorf("cannot write to %s file", gitHubEnvName)
	}
	defer f.Close()

	for k, v := range vars {
		if err = e.maskValue(v); err != nil {
			return err
		}
		if _, err = f.WriteString(e.prepareKeyValueMessage(k, v)); err != nil {
			return err
		}
	}

	return nil
}

var newLineRegex = regexp.MustCompile("\r\n|\r|\n")

// prepareKeyValueMessage formats the key-value pair for the GITHUB_ENV file.
// See also: https://github.com/actions/toolkit/blob/main/packages/core/src/file-command.ts#L27
func (e *GithubActionsExporter) prepareKeyValueMessage(key string, value string) string {
	delimiter := fmt.Sprintf("ghadelimiter_%s", e.uuidGenerator())
	return fmt.Sprintf("%s<<%s%s%s%s%s%s", key, delimiter, e.eol, value, e.eol, delimiter, e.eol)
}

func (e *GithubActionsExporter) maskValue(value string) error {
	lines := newLineRegex.Split(value, -1)
	ctx := context.Background()

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		opts := &execext.RunCommandOptions{
			Command: fmt.Sprintf(`echo ::add-mask::"%s"`, line),
			Stdout:  os.Stdout,
		}

		if err := execext.RunCommand(ctx, opts); err != nil {
			return err
		}
	}

	return nil
}
