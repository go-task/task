package taskfile

type (
	NodeOption interface {
		ApplyToBaseNode(n *baseNode)
	}
	// baseNode is a generic node that implements the Parent() methods of the
	// NodeReader interface. It does not implement the Read() method and it
	// designed to be embedded in other node types so that this boilerplate code
	// does not need to be repeated.
	baseNode struct {
		parent        Node
		dir           string
		checksum      string
		caCert        string
		cert          string
		certKey       string
		headersByHost HeadersByHost
	}
)

func NewBaseNode(dir string, opts ...NodeOption) *baseNode {
	node := &baseNode{
		parent: nil,
		dir:    dir,
	}

	// Apply options
	for _, opt := range opts {
		opt.ApplyToBaseNode(node)
	}

	return node
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
