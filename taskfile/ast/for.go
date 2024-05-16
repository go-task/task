package ast

import (
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

type For struct {
	From  string
	List  []any
	Var   string
	Split string
	As    string
}

func (f *For) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var from string
		if err := node.Decode(&from); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		f.From = from
		return nil

	case yaml.SequenceNode:
		var list []any
		if err := node.Decode(&list); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		f.List = list
		return nil

	case yaml.MappingNode:
		var forStruct struct {
			Var   string
			Split string
			As    string
		}
		if err := node.Decode(&forStruct); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		if forStruct.Var == "" {
			return errors.NewTaskfileDecodeError(nil, node).WithMessage("invalid keys in for")
		}
		f.Var = forStruct.Var
		f.Split = forStruct.Split
		f.As = forStruct.As
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("for")
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
