package taskfile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sync/singleflight"

	"github.com/go-task/task/v3/internal/execext"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/orderedmap"
)

// Vars is a string[string] variables map.
type Vars struct {
	orderedmap.OrderedMap[string, Var]
}

// ToCacheMap converts Vars to a map containing only the static
// variables
func (vs *Vars) ToCacheMap() (m map[string]any) {
	m = make(map[string]any, vs.Len())
	_ = vs.Range(func(k string, v Var) error {
		if v.Sh != "" {
			// Dynamic variable is not yet resolved; trigger
			// <no value> to be used in templates.
			return nil
		}

		if v.Live != nil {
			m[k] = v.Live
		} else if v.Lazy != nil {
			m[k] = v.Lazy
		} else {
			m[k] = v.Static
		}
		return nil
	})
	return
}

// Wrapper around OrderedMap.Set to ensure we don't get nil pointer errors
func (vs *Vars) Range(f func(k string, v Var) error) error {
	if vs == nil {
		return nil
	}
	return vs.OrderedMap.Range(f)
}

// Wrapper around OrderedMap.Merge to ensure we don't get nil pointer errors
func (vs *Vars) Merge(other *Vars) {
	if vs == nil || other == nil {
		return
	}
	vs.OrderedMap.Merge(other.OrderedMap)
}

// Wrapper around OrderedMap.Len to ensure we don't get nil pointer errors
func (vs *Vars) Len() int {
	if vs == nil {
		return 0
	}
	return vs.OrderedMap.Len()
}

// DeepCopy creates a new instance of Vars and copies
// data by value from the source struct.
func (vs *Vars) DeepCopy() *Vars {
	if vs == nil {
		return nil
	}
	return &Vars{
		OrderedMap: vs.OrderedMap.DeepCopy(),
	}
}

// Var represents either a static or dynamic variable.
type Var struct {
	Static string
	Live   any
	Sh     string
	Dir    string
	Lazy   *LazySh
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var str string
		if err := node.Decode(&str); err != nil {
			return err
		}
		v.Static = str
		return nil

	case yaml.MappingNode:
		var sh struct {
			Sh   string
			Lazy bool
		}
		if err := node.Decode(&sh); err != nil {
			return err
		}
		if sh.Lazy {
			v.Lazy = &LazySh{
				Sh:     sh.Sh,
				Sf:     &singleflight.Group{},
				Stderr: os.Stderr,
			}
			return nil
		}
		v.Sh = sh.Sh
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into variable", node.Line, node.ShortTag())
}

type LazySh struct {
	Sh     string
	Dir    string
	Val    string
	Done   bool
	Sf     *singleflight.Group
	Stderr io.Writer
}

func (l *LazySh) String() string {
	val, err, _ := l.Sf.Do("", func() (any, error) {
		if l.Done {
			return l.Val, nil
		}

		var stdout bytes.Buffer
		opts := &execext.RunCommandOptions{
			Command: l.Sh,
			Dir:     l.Dir,
			Stdout:  &stdout,
			Stderr:  l.Stderr,
		}
		if err := execext.RunCommand(context.Background(), opts); err != nil {
			return "", fmt.Errorf(`task: Command "%s" failed: %s`, opts.Command, err)
		}
		result := strings.TrimSuffix(stdout.String(), "\r\n")
		result = strings.TrimSuffix(result, "\n")
		l.Val = result
		l.Done = true
		return result, nil
	})
	if err != nil {
		panic(err)
	}
	return val.(string)
}
