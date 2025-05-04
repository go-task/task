package taskfile

type (
	NodeOption func(*baseNode)
	// baseNode is a generic node that implements the Parent() methods of the
	// NodeReader interface. It does not implement the Read() method and it
	// designed to be embedded in other node types so that this boilerplate code
	// does not need to be repeated.
	baseNode struct {
		parent   Node
		dir      string
		checksum string
	}
)

func NewBaseNode(dir string, opts ...NodeOption) *baseNode {
	node := &baseNode{
		parent: nil,
		dir:    dir,
	}

	// Apply options
	for _, opt := range opts {
		opt(node)
	}

	return node
}

func WithParent(parent Node) NodeOption {
	return func(node *baseNode) {
		node.parent = parent
	}
}

func WithChecksum(checksum string) NodeOption {
	return func(node *baseNode) {
		node.checksum = checksum
	}
}

func (node *baseNode) Parent() Node {
	return node.parent
}

func (node *baseNode) Dir() string {
	return node.dir
}

func (node *baseNode) Checksum() string {
	return node.checksum
}

func (node *baseNode) Verify(checksum string) bool {
	return node.checksum == "" || node.checksum == checksum
}
