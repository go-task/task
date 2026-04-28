package execext_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/execext"
)

func TestRunCommandEscapedBraces(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := execext.RunCommand(context.Background(), &execext.RunCommandOptions{
		Command: `printf '<%s>\n' \{\{iriname\}\}`,
		Stdout:  &stdout,
	})

	require.NoError(t, err)
	require.Equal(t, "<{{iriname}}>\n", stdout.String())
}
