package dockercomposeiface

import (
	"context"
	"fmt"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	cw_types "github.com/go-task/task/v3/internal/containerwrapper/types"
	"github.com/sirupsen/logrus"
)

type DockerComposeWrapper struct {
	srv       api.Service
	dockerCLI *command.DockerCli
	project   *types.Project
}

func CreateDockerComposeWrapper(dockerContext string, project *types.Project) (DockerComposeWrapper, error) {
	fmt.Println("Currently level:", logrus.GetLevel())

	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return DockerComposeWrapper{}, err
	}

	//Magic line to fix error:
	//Failed to initialize: unable to resolve dockercomposeiface endpoint: no context store initialized
	myOpts := &flags.ClientOptions{Context: dockerContext, LogLevel: "error"}
	err = dockerCli.Initialize(myOpts)
	if err != nil {
		return DockerComposeWrapper{}, err
	}

	srv := compose.NewComposeService(dockerCli)
	fmt.Println("Currently level:", logrus.GetLevel())
	return DockerComposeWrapper{srv, dockerCli, project}, nil
}

func (dcw DockerComposeWrapper) Down(ctx context.Context) error {
	return dcw.srv.Down(ctx, dcw.project.Name, api.DownOptions{})
}

func (dcw DockerComposeWrapper) Up(ctx context.Context) error {
	return dcw.srv.Up(ctx, dcw.project, api.UpOptions{})
}

func (dcw DockerComposeWrapper) Exec(ctx context.Context, execOpts cw_types.ExecOptions) (int, error) {

	runOptions := api.RunOptions{
		Command:     []string{"/bin/bash", "-c", execOpts.Command},
		WorkingDir:  execOpts.Workdir,
		User:        execOpts.User,
		Environment: execOpts.Env,
		Service:     execOpts.Service,
		Tty:         true,
	}

	return dcw.srv.Exec(ctx, dcw.project.Name, runOptions)
}
