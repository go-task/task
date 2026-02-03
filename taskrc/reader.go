package taskrc

import (
	"os"

	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/taskrc/ast"
)

type (
	// DebugFunc is a function that can be called to log debug messages.
	DebugFunc func(string)
	// A ReaderOption is any type that can apply a configuration to a [Reader].
	ReaderOption interface {
		ApplyToReader(*Reader)
	}
	// A Reader will recursively read Taskfiles from a given [Node] and build a
	// [ast.TaskRC] from them.
	Reader struct {
		debugFunc DebugFunc
	}
)

// NewReader constructs a new Taskfile [Reader] using the given Node and
// options.
func NewReader(opts ...ReaderOption) *Reader {
	r := &Reader{
		debugFunc: nil,
	}
	r.Options(opts...)
	return r
}

// Options loops through the given [ReaderOption] functions and applies them to
// the [Reader].
func (r *Reader) Options(opts ...ReaderOption) {
	for _, opt := range opts {
		opt.ApplyToReader(r)
	}
}

// WithDebugFunc sets the debug function to be used by the [Reader]. If set,
// this function will be called with debug messages. This can be useful if the
// caller wants to log debug messages from the [Reader]. By default, no debug
// function is set and the logs are not written.
func WithDebugFunc(debugFunc DebugFunc) ReaderOption {
	return &debugFuncOption{debugFunc: debugFunc}
}

type debugFuncOption struct {
	debugFunc DebugFunc
}

func (o *debugFuncOption) ApplyToReader(r *Reader) {
	r.debugFunc = o.debugFunc
}

// Read will read the Task config defined by the [Reader]'s [Node].
func (r *Reader) Read(node *Node) (*ast.TaskRC, error) {
	var config ast.TaskRC

	if node == nil {
		return nil, os.ErrInvalid
	}

	// Read the file
	b, err := os.ReadFile(node.entrypoint)
	if err != nil {
		return nil, err
	}

	// Parse the content
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
