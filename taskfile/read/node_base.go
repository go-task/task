package read

type (
	NodeOption func(*BaseNode)
	// BaseNode is a generic node that implements the Parent() and Optional()
	// methods of the NodeReader interface. It does not implement the Read() method
	// and it designed to be embedded in other node types so that this boilerplate
	// code does not need to be repeated.
	BaseNode struct {
		parent   Node
		optional bool
	}
)

func NewBaseNode(opts ...NodeOption) *BaseNode {
	node := &BaseNode{
		parent:   nil,
		optional: false,
	}

	// Apply options
	for _, opt := range opts {
		opt(node)
	}

	return node
}

func WithParent(parent Node) NodeOption {
	return func(node *BaseNode) {
		node.parent = parent
	}
}

func (node *BaseNode) Parent() Node {
	return node.parent
}

func WithOptional(optional bool) NodeOption {
	return func(node *BaseNode) {
		node.optional = optional
	}
}

func (node *BaseNode) Optional() bool {
	return node.optional
}
