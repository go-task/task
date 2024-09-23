package taskfile_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile"
)

func TestNewCache(t *testing.T) {
	testCases := map[string]struct {
		options []taskfile.CacheOption
	}{
		"no options set": {},

		"TTL option set": {
			options: []taskfile.CacheOption{taskfile.WithTTL(time.Hour)},
		},
	}

	for desc, testCase := range testCases {
		t.Run(desc, func(t *testing.T) {
			_, err := taskfile.NewCache(t.TempDir(), testCase.options...)
			require.NoError(t, err, "creating new cache")
		})
	}
}
