package read

// BaseNode is a generic node that implements the Parent() and Optional()
// methods of the NodeReader interface. It does not implement the Read() method
// and it designed to be embedded in other node types so that this boilerplate
// code does not need to be repeated.
type BaseNode struct {
	parent   Node
	optional bool
}

func (node *BaseNode) Parent() Node {
	return node.parent
}

func (node *BaseNode) Optional() bool {
	return node.optional
}
