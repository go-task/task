package taskfile

import (
	"bufio"
	"context"
	"fmt"
	"os"
)

// A StdinNode is a node that reads a taskfile from the standard input stream.
type StdinNode struct {
	*BaseNode
	Dir string
}

func NewStdinNode(dir string) (*StdinNode, error) {
	base := NewBaseNode()
	return &StdinNode{
		BaseNode: base,
		Dir:      dir,
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

func (node *StdinNode) BaseDir() string {
	return node.Dir
}
