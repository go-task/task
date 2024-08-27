package taskfile

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// A StdinNode is a node that reads a taskfile from the standard input stream.
type StdinNode struct {
	*BaseNode
}

func NewStdinNode(dir string) (*StdinNode, error) {
	return &StdinNode{
		BaseNode: NewBaseNode(dir),
	}, nil
}

func (node *StdinNode) Location() string {
	return "__stdin__"
}

func (node *StdinNode) Read(ctx context.Context) (*source, error) {
	var stdin []byte
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		stdin = fmt.Appendln(stdin, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &source{
		FileContent: stdin,
		Filename:    node.Location(),
	}, nil
}

func (node *StdinNode) ResolveEntrypoint(entrypoint string) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if strings.Contains(entrypoint, "://") {
		return entrypoint, nil
	}

	path, err := execext.Expand(entrypoint)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	return filepathext.SmartJoin(node.Dir(), path), nil
}

func (node *StdinNode) ResolveDir(dir string) (string, error) {
	path, err := execext.Expand(dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	return filepathext.SmartJoin(node.Dir(), path), nil
}

func (node *StdinNode) FilenameAndLastDir() (string, string) {
	return "", "__stdin__"
}
