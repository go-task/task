package taskfile

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile/ast"
)

// A StdinNode is a node that reads a taskfile from the standard input stream.
type StdinNode struct {
	*BaseNode
}

func NewStdinNode(dir string) (*StdinNode, error) {
	base := NewBaseNode()
	base.dir = dir
	return &StdinNode{
		BaseNode: base,
	}, nil
}

func (node *StdinNode) Location() string {
	return "__stdin__"
}

func (node *StdinNode) Remote() bool {
	return false
}

func (node *StdinNode) Read(ctx context.Context) ([]byte, error) {
	var stdin []byte
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		stdin = fmt.Appendln(stdin, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stdin, nil
}

func (node *StdinNode) ResolveIncludeEntrypoint(include ast.Include) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if strings.Contains(include.Taskfile, "://") {
		return include.Taskfile, nil
	}

	path, err := execext.Expand(include.Taskfile)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	return filepathext.SmartJoin(node.Dir(), path), nil
}

func (node *StdinNode) ResolveIncludeDir(include ast.Include) (string, error) {
	path, err := execext.Expand(include.Dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	return filepathext.SmartJoin(node.Dir(), path), nil
}
