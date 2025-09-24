package taskfile

import (
	"bufio"
	"fmt"
	"os"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// A StdinNode is a node that reads a taskfile from the standard input stream.
type StdinNode struct {
	*baseNode
}

func NewStdinNode(dir string) (*StdinNode, error) {
	return &StdinNode{
		baseNode: NewBaseNode(dir),
	}, nil
}

func (node *StdinNode) Location() string {
	return "__stdin__"
}

func (node *StdinNode) Remote() bool {
	return false
}

func (node *StdinNode) Read() ([]byte, error) {
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

func (node *StdinNode) ResolveEntrypoint(entrypoint string) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if isRemoteEntrypoint(entrypoint) {
		return entrypoint, nil
	}

	path, err := execext.ExpandLiteral(entrypoint)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	return filepathext.SmartJoin(node.Dir(), path), nil
}

func (node *StdinNode) ResolveDir(dir string) (string, error) {
	path, err := execext.ExpandLiteral(dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	return filepathext.SmartJoin(node.Dir(), path), nil
}
