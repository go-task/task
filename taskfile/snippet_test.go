package taskfile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const sample = `version: 3

tasks:
  default:
    vars:
      FOO: foo
      BAR: bar
    cmds:
      - echo "{{.FOO}}"
      - echo "{{.BAR}}"
`

func TestNewSnippet(t *testing.T) {
	tests := []struct {
		name    string
		b       []byte
		line    int
		column  int
		padding int
		want    *Snippet
	}{
		{
			name:    "first line, first column",
			b:       []byte(sample),
			line:    1,
			column:  1,
			padding: 0,
			want: &Snippet{
				lines: []string{
					"\x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
				},
				start:   1,
				end:     1,
				line:    1,
				column:  1,
				padding: 0,
			},
		},
		{
			name:    "first line, first column, padding=2",
			b:       []byte(sample),
			line:    1,
			column:  1,
			padding: 2,
			want: &Snippet{
				lines: []string{
					"\x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
					"\x1b[1m\x1b[30m\x1b[0m",
					"\x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
				},
				start:   1,
				end:     3,
				line:    1,
				column:  1,
				padding: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSnippet(tt.b, tt.line, tt.column, tt.padding)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSnippetString(t *testing.T) {
	tests := []struct {
		name    string
		b       []byte
		line    int
		column  int
		padding int
		want    string
	}{
		{
			name:    "empty",
			b:       []byte{},
			line:    1,
			column:  1,
			padding: 0,
			want:    "",
		},
		{
			name:    "0th line, 0th column (no indicators)",
			b:       []byte(sample),
			line:    0,
			column:  0,
			padding: 0,
			want:    "",
		},
		{
			name:    "1st line, 0th column (line indicator only)",
			b:       []byte(sample),
			line:    1,
			column:  0,
			padding: 0,
			want:    "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name:    "0th line, 1st column (column indicator only)",
			b:       []byte(sample),
			line:    0,
			column:  1,
			padding: 0,
			want:    "",
		},
		{
			name:    "0th line, 1st column, padding=2 (column indicator only)",
			b:       []byte(sample),
			line:    0,
			column:  1,
			padding: 2,
			want:    "  1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  2 | \x1b[1m\x1b[30m\x1b[0m\n    | ^",
		},
		{
			name:    "1st line, 1st column",
			b:       []byte(sample),
			line:    1,
			column:  1,
			padding: 0,
			want:    "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    | ^",
		},
		{
			name:    "1st line, 10th column",
			b:       []byte(sample),
			line:    1,
			column:  10,
			padding: 0,
			want:    "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |          ^",
		},
		{
			name:    "1st line, 1st column, padding=2",
			b:       []byte(sample),
			line:    1,
			column:  1,
			padding: 2,
			want:    "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    | ^\n  2 | \x1b[1m\x1b[30m\x1b[0m\n  3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name:    "1st line, 10th column, padding=2",
			b:       []byte(sample),
			line:    1,
			column:  10,
			padding: 2,
			want:    "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |          ^\n  2 | \x1b[1m\x1b[30m\x1b[0m\n  3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name:    "5th line, 1st column",
			b:       []byte(sample),
			line:    5,
			column:  1,
			padding: 0,
			want:    "> 5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    | ^",
		},
		{
			name:    "5th line, 5th column",
			b:       []byte(sample),
			line:    5,
			column:  5,
			padding: 0,
			want:    "> 5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |     ^",
		},
		{
			name:    "5th line, 5th column, padding=2",
			b:       []byte(sample),
			line:    5,
			column:  5,
			padding: 2,
			want:    "  3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  4 | \x1b[1m\x1b[30m  \x1b[0m\x1b[33mdefault\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n> 5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |     ^\n  6 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mFOO\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mfoo\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  7 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mBAR\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mbar\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name:    "10th line, 1st column",
			b:       []byte(sample),
			line:    10,
			column:  1,
			padding: 0,
			want:    "> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     | ^",
		},
		{
			name:    "10th line, 23rd column",
			b:       []byte(sample),
			line:    10,
			column:  23,
			padding: 0,
			want:    "> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     |                       ^",
		},
		{
			name:    "10th line, 24th column (out of bounds)",
			b:       []byte(sample),
			line:    10,
			column:  24,
			padding: 0,
			want:    "> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     |                        ^",
		},
		{
			name:    "10th line, 23rd column, padding=2",
			b:       []byte(sample),
			line:    10,
			column:  23,
			padding: 2,
			want:    "   8 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mcmds\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   9 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.FOO}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     |                       ^",
		},
		{
			name:    "5th line, 5th column, padding=100",
			b:       []byte(sample),
			line:    5,
			column:  5,
			padding: 100,
			want:    "   1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   2 | \x1b[1m\x1b[30m\x1b[0m\n   3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   4 | \x1b[1m\x1b[30m  \x1b[0m\x1b[33mdefault\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n>  5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     |     ^\n   6 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mFOO\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mfoo\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   7 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mBAR\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mbar\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   8 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mcmds\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   9 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.FOO}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name:    "11th line (out of bounds), 1st column",
			b:       []byte(sample),
			line:    11,
			column:  1,
			padding: 0,
			want:    "",
		},
		{
			name:    "11th line (out of bounds), 1st column, padding=2",
			b:       []byte(sample),
			line:    11,
			column:  1,
			padding: 2,
			want:    "   9 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.FOO}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snippet := NewSnippet(tt.b, tt.line, tt.column, tt.padding)
			got := snippet.String()
			if strings.Contains(got, "\t") {
				t.Fatalf("tab character found in snippet - check the sample string")
			}
			require.Equal(t, tt.want, got)
		})
	}
}
