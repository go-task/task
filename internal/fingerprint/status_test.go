package fingerprint

import (
	"context"
	"testing"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

func testLogger() *logger.Logger {
	return &logger.Logger{Color: false}
}

func TestStatusCheckerHonorsPipefail(t *testing.T) {
	task := &ast.Task{
		Dir:    t.TempDir(),
		Status: []string{"cat does-not-exist 2>/dev/null | true"},
	}

	withoutPipefail := NewStatusChecker(testLogger(), nil, nil)
	upToDate, err := withoutPipefail.IsUpToDate(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}
	if !upToDate {
		t.Fatal("without pipefail a pipeline whose last command succeeds should be up to date")
	}

	withPipefail := NewStatusChecker(testLogger(), []string{"pipefail"}, nil)
	upToDate, err = withPipefail.IsUpToDate(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}
	if upToDate {
		t.Fatal("with pipefail a failing pipeline component should mark the task as not up to date")
	}
}
