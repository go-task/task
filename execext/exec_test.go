// execext_test.go
package execext

import (
	"context"
	"io/ioutil"
	"strings"
	"sync"
	"testing"

	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/syntax"
)

func BenchmarkNoPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f, err := syntax.NewParser().Parse(strings.NewReader(`echo "Hello, World!"`), "")
		if err != nil {
			panic(err)
		}
		r := interp.Runner{
			Context: context.TODO(),
			Stdout:  ioutil.Discard,
			Stderr:  ioutil.Discard,
		}
		if err = r.Reset(); err != nil {
			panic(err)
		}
		if err = r.Run(f); err != nil {
			panic(err)
		}
	}
}

func BenchmarkPool(b *testing.B) {
	parserPool := sync.Pool{
		New: func() interface{} {
			return syntax.NewParser()
		},
	}
	runnerPool := sync.Pool{
		New: func() interface{} {
			return &interp.Runner{}
		},
	}

	for i := 0; i < b.N; i++ {
		parser := parserPool.Get().(*syntax.Parser)
		defer parserPool.Put(parser)

		f, err := parser.Parse(strings.NewReader(`echo "Hello, World!"`), "")
		if err != nil {
			panic(err)
		}

		r := runnerPool.Get().(*interp.Runner)
		defer runnerPool.Put(r)

		r.Stdout = ioutil.Discard
		r.Stderr = ioutil.Discard

		if err = r.Reset(); err != nil {
			panic(err)
		}
		if err = r.Run(f); err != nil {
			panic(err)
		}
	}
}
