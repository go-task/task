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
	// Resolve to entrypoint without adjustment.
	if isRemoteEntrypoint(entrypoint) {
		return entrypoint, nil
	}
	// Resolve relative to this node.Dir() (i.e. the root node), or absolute.
	entrypoint, err := execext.ExpandLiteral(entrypoint)
	if err != nil {
		return "", err
	}
	return filepathext.SmartJoin(node.Dir(), entrypoint), nil
}

func (node *StdinNode) ResolveDir(dir string) (string, error) {
	if len(dir) == 0 {
		// Resolve to the current node.Dir().
		return node.Dir(), nil
	} else {
		// Resolve include.Dir, relative to this node.Dir(), or absolute.
		dir, err := execext.ExpandLiteral(dir)
		if err != nil {
			return "", err
		}
		return filepathext.SmartJoin(node.Dir(), dir), nil
	}
}
