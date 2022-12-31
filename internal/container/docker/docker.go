package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/go-task/task/v3/internal/execext"
)

type Docker struct {
	DockerCli string
	Image     string
	Flags     []string
	Env       map[string]string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (d *Docker) Setup() error {
	if d.DockerCli == "" {
		d.DockerCli = "docker"
	}
	return nil
}

func (d *Docker) Exec(ctx context.Context, cmd string) error {
	dockerArgs := []string{"run"}

	for k, v := range d.Env {
		dockerArgs = append(dockerArgs, "--env", fmt.Sprintf("%s=%s", k, v))
	}

	dockerArgs = append(dockerArgs, d.Flags...)
	dockerArgs = append(dockerArgs, d.Image, "sh", "-c", cmd)

	shCmd, err := execext.Build(d.DockerCli, dockerArgs...)
	if err != nil {
		return err
	}

	fmt.Printf("DOCKER: %s\n", shCmd)

	return execext.RunCommand(ctx, &execext.RunCommandOptions{
		Command: shCmd,
		Dir:     "", // TODO(@andreynering): Implement this
		Stdin:   d.Stdin,
		Stdout:  d.Stdout,
		Stderr:  d.Stderr,
	})
}
