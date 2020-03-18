package execext

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

// RunCommandOptions is the options for the RunCommand func
type RunCommandOptions struct {
	Image   string
	Volumes []string
	Command string
	Dir     string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

var (
	// ErrNilOptions is returned when a nil options is given
	ErrNilOptions = errors.New("execext: nil options given")
)

// RunCommand runs a shell command
func RunCommand(ctx context.Context, opts *RunCommandOptions) error {
	if opts == nil {
		return ErrNilOptions
	}

	if opts.Image != "" {
		return RunCommandInDocker(ctx, opts)
	}

	p, err := syntax.NewParser().Parse(strings.NewReader(opts.Command), "")
	if err != nil {
		return err
	}

	environ := opts.Env
	if len(environ) == 0 {
		environ = os.Environ()
	}

	r, err := interp.New(
		interp.Dir(opts.Dir),
		interp.Env(expand.ListEnviron(environ...)),

		interp.OpenHandler(func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
			if path == "/dev/null" {
				return devNull{}, nil
			}
			return interp.DefaultOpenHandler()(ctx, path, flag, perm)
		}),

		interp.StdIO(opts.Stdin, opts.Stdout, opts.Stderr),
	)
	if err != nil {
		return err
	}
	err = r.Run(ctx, p)

	return err
}

func RunCommandInDocker(ctx context.Context, opts *RunCommandOptions) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	absoluteDir, _ := filepath.Abs(opts.Dir)
	image, err := reference.ParseNormalizedNamed(opts.Image)
	if err != nil {
		return err
	}

	pullReader, err := cli.ImagePull(context.Background(), image.String(), types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer pullReader.Close()

	termFd, isTerminal := term.GetFdInfo(os.Stdout)
	err = jsonmessage.DisplayJSONMessagesStream(pullReader, os.Stdout, termFd, isTerminal, nil)
	if err != nil {
		return err
	}

	var mounts []mount.Mount
	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: absoluteDir,
		Target: absoluteDir,
	})

	for _, volume := range opts.Volumes {
		volumePaths := strings.Split(volume, ":")

		var readOnly = false
		if len(volumePaths) == 3 {
			readOnly = true
		} else if len(volumePaths) != 2 {
			return errors.New(fmt.Sprintf("invalid volume \"%s\"", volume))
		}

		localPath := volumePaths[0]
		if !filepath.IsAbs(localPath) {
			localPath = filepath.Join(absoluteDir, localPath)
		}

		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   localPath,
			Target:   volumePaths[1],
			ReadOnly: readOnly,
		})
	}

	cont, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:        image.String(),
			Cmd:          []string{"-c", opts.Command},
			Entrypoint:   []string{"/bin/sh"},
			WorkingDir:   absoluteDir,
			AttachStdout: true,
			AttachStderr: true,
			AttachStdin:  true,
		},
		&container.HostConfig{
			Mounts: mounts,
		}, nil, "")

	if err != nil {
		return err
	}

	err = cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	reader, err := cli.ContainerLogs(context.Background(), cont.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(opts.Stdout, reader)
	if err != nil {
		return err
	}

	code, err := cli.ContainerWait(context.Background(), cont.ID)
	if code != 0 {
		return errors.New(fmt.Sprintf("exit status %v", code))
	}

	if err != nil {
		return err
	}

	return nil
}

// IsExitError returns true the given error is an exis status error
func IsExitError(err error) bool {
	if _, ok := interp.IsExitStatus(err); ok {
		return true
	}
	return false
}

// Expand is a helper to mvdan.cc/shell.Fields that returns the first field
// if available.
func Expand(s string) (string, error) {
	s = filepath.ToSlash(s)
	s = strings.Replace(s, " ", `\ `, -1)
	fields, err := shell.Fields(s, nil)
	if err != nil {
		return "", err
	}
	if len(fields) > 0 {
		return fields[0], nil
	}
	return "", nil
}
