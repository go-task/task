package types

import (
	"context"
)

/*
ContainerInterface wraps the backend container implementation,
the idea here being that any form of containerization could be
used: docker, docker compose, podman, etc
Currently only docker-compose is supported.

Up:  Start all the required containers
Down: Stop, shutdown, etc all the required containers
Exec: Execute a command inside an already running container
*/
type ContainerInterface interface {
	Up(context.Context) error
	Down(context.Context) error
	Exec(context.Context, ExecOptions) (int, error)
}

type ExecOptions struct {
	Command string
	Workdir string
	Env     []string
	User    string
	Service string // Name of container to execute the commands in.
}
