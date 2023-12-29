package containerwrapper

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/compose-spec/compose-go/loader"
	compose_types "github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/go-task/task/v3/internal/containerwrapper/dockercomposeiface"
	"github.com/go-task/task/v3/internal/containerwrapper/types"
	"github.com/go-task/task/v3/internal/logger"
	"gopkg.in/yaml.v3"
	"strings"
)

/*
The input into the ContainerSetup is a valid docker compose definition.
The input is parsed and converted into a project, this has a few advantages:
- The input required by the end user is a well known format
- There is lots of documentation on the format
- All the parsing and validation is done by compose-go lib.
- The implementation is handled by the ContainerInterface, currently only
docker compose is supported.
*/

type ContainerSetup struct {
	project *compose_types.Project
	context string // Docker context reference
	CI      types.ContainerInterface
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// Using compose-go to handle the parsing of the contents.
// Parsing all the components especially the volumes could be problematic
// instead we farm all that out to the compose-go loader package which
// is designed to parse, load and validate the spec/schema.
// docker compose might be overkill for what is required however
// this helps provide a standard and consistent layout
func (tdc *ContainerSetup) UnmarshalYAML(node *yaml.Node) error {
	ctx := context.TODO()

	tdc.context = "default"

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == "context" {
			var str string
			if err := valueNode.Decode(&str); err != nil {
				return err
			}
			tdc.context = str
		}

		if keyNode.Value == "definition" {
			b, err := yaml.Marshal(valueNode)
			if err != nil {
				return err
			}
			hasher := sha1.New()
			hasher.Write(b)
			// The loader will try and generate a project name based on the file path
			// however there is no real file because we are loading data via []bytes
			// We generate and force a project name
			projectName := hex.EncodeToString(hasher.Sum(nil))
			configDetails := compose_types.ConfigDetails{
				WorkingDir: "/taskfile/in-memory/",
				ConfigFiles: []compose_types.ConfigFile{
					{Filename: "docker-compose.yaml", Content: b},
				},
				Environment: nil,
			}
			p, err := loader.LoadWithContext(ctx, configDetails, func(options *loader.Options) {
				options.SetProjectName(projectName, true)
			})
			if err != nil {
				return err
			}
			addServiceLabels(p)
			tdc.project = p
		}
	}

	if tdc.project == nil {
		return fmt.Errorf("containerwrapper missing project")
	}

	if len(tdc.project.Services) < 1 {
		return fmt.Errorf("containerwrapper node missing definition, expected at least 1 service definition")
	}
	// We don't create the ContainerInterface (tdc.CI) yet because that is only required if a task is being
	// executed inside a container. At this point only a configuration has been detected and loaded.
	// An example could be in a CI pipeline all tasks are executed in a container but not required when run locally.
	return nil
}

// Initialize is responsible for creating the ContainerInterface and
// running any form of container required initialization. It does not have to
// start the containers.
// If should be safe to execute multiple times.
func (tdc *ContainerSetup) Initialize() error {
	// Already initialized
	if tdc.CI != nil {
		return nil
	}
	containerInterface, err := dockercomposeiface.CreateDockerComposeWrapper(tdc.context, tdc.project)
	if err != nil {
		return err
	}

	tdc.CI = containerInterface
	return nil
}

// Cleanup should handle all the shutdown required work
// such as stopping containers or calling docker compose down.
// It should be safe to call repeatedly, i.e be idempotent and
// safe to call even if ContainerSetup has never been initialized
func (tdc *ContainerSetup) Cleanup(ctx context.Context) error {
	// Never initialized therefore nothing to clean up.
	if tdc.CI == nil {
		return nil
	}
	return tdc.CI.Down(ctx)
}

// createBaseExecOptions creates an initial set of options
// that all container execs might require.
func (tdc *ContainerSetup) createBaseExecOptions() types.ExecOptions {
	firstService := tdc.project.Services[0]
	execOpt := types.ExecOptions{
		Service: firstService.Name,
		Workdir: firstService.WorkingDir,
		User:    firstService.User,
	}
	return execOpt
}

func (tdc *ContainerSetup) Exec(ctx context.Context, cmd string, envVars []string, workDir string, log *logger.Logger) error {
	execOpts := tdc.createBaseExecOptions()
	execOpts.Command = cmd
	execOpts.Env = envVars
	if strings.HasPrefix(workDir, "/") {
		execOpts.Workdir = workDir
	} else {
		log.VerboseErrf(logger.Yellow, "task: [%s] dir: %s needs to be absolute path when using container, "+
			"using container default WORKDIR: %s\n", "t.Name()", workDir, execOpts.Workdir)
	}

	result, err := tdc.CI.Exec(ctx, execOpts)
	if result != 0 {
		err = fmt.Errorf("task: command [%s] returned %d, non-zero exit status from within container\n", cmd, result)
	}
	return err
}

/*
addServiceLabels adds the labels docker compose expects to exist on services.
This is required for future compose operations to work, such as finding
containers that are part of a service.
*/
func addServiceLabels(project *compose_types.Project) {
	for i, s := range project.Services {
		s.CustomLabels = map[string]string{
			api.ProjectLabel:     project.Name,
			api.ServiceLabel:     s.Name,
			api.VersionLabel:     api.ComposeVersion,
			api.WorkingDirLabel:  "/",
			api.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			api.OneoffLabel:      "False", // default, will be overridden by `run` command
		}
		project.Services[i] = s
	}
}
