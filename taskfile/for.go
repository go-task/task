package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/deepcopy"
)

type For struct {
	From  string
	List  []string
	Var   string
	Split string
	As    string
}

func (f *For) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var from string
		if err := node.Decode(&from); err != nil {
			return err
		}
		f.From = from
		return nil

	case yaml.SequenceNode:
		var list []string
		if err := node.Decode(&list); err != nil {
			return err
		}
		f.List = list
		return nil

	case yaml.MappingNode:
		var forStruct struct {
			Var   string
			Split string
			As    string
		}
		if err := node.Decode(&forStruct); err == nil && forStruct.Var != "" {
			f.Var = forStruct.Var
			f.Split = forStruct.Split
			f.As = forStruct.As
			return nil
		}

		return fmt.Errorf("yaml: line %d: invalid keys in for", node.Line)
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into for", node.Line, node.ShortTag())
}

func (f *For) DeepCopy() *For {
	if f == nil {
		return nil
	}
	return &For{
		From:  f.From,
		List:  deepcopy.Slice(f.List),
		Var:   f.Var,
		Split: f.Split,
		As:    f.As,
	}
}
