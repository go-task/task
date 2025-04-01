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
	t.Parallel()
	tests := []struct {
		name string
		b    []byte
		opts []SnippetOption
		want *Snippet
	}{
		{
			name: "first line, first column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(1),
				WithColumn(1),
			},
			want: &Snippet{
				linesRaw: []string{
					"version: 3",
				},
				linesHighlighted: []string{
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
			name: "first line, first column, padding=2",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(1),
				WithColumn(1),
				WithPadding(2),
			},
			want: &Snippet{
				linesRaw: []string{
					"version: 3",
					"",
					"tasks:",
				},
				linesHighlighted: []string{
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
			t.Parallel()
			got := NewSnippet(tt.b, tt.opts...)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSnippetString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		b    []byte
		opts []SnippetOption
		want string
	}{
		{
			name: "empty",
			b:    []byte{},
			opts: []SnippetOption{
				WithLine(1),
				WithColumn(1),
			},
			want: "",
		},
		{
			name: "0th line, 0th column (no indicators)",
			b:    []byte(sample),
			want: "",
		},
		{
			name: "1st line, 0th column (line indicator only)",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(1),
			},
			want: "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name: "0th line, 1st column (column indicator only)",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithColumn(1),
			},
			want: "",
		},
		{
			name: "0th line, 1st column, padding=2 (column indicator only)",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithColumn(1),
				WithPadding(2),
			},
			want: "  1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  2 | \x1b[1m\x1b[30m\x1b[0m\n    | ^",
		},
		{
			name: "1st line, 1st column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(1),
				WithColumn(1),
			},
			want: "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    | ^",
		},
		{
			name: "1st line, 10th column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(1),
				WithColumn(10),
			},
			want: "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |          ^",
		},
		{
			name: "1st line, 1st column, padding=2",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(1),
				WithColumn(1),
				WithPadding(2),
			},
			want: "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    | ^\n  2 | \x1b[1m\x1b[30m\x1b[0m\n  3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name: "1st line, 10th column, padding=2",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(1),
				WithColumn(10),
				WithPadding(2),
			},
			want: "> 1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |          ^\n  2 | \x1b[1m\x1b[30m\x1b[0m\n  3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name: "5th line, 1st column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(5),
				WithColumn(1),
			},
			want: "> 5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    | ^",
		},
		{
			name: "5th line, 5th column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(5),
				WithColumn(5),
			},
			want: "> 5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |     ^",
		},
		{
			name: "5th line, 5th column, padding=2",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(5),
				WithColumn(5),
				WithPadding(2),
			},
			want: "  3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  4 | \x1b[1m\x1b[30m  \x1b[0m\x1b[33mdefault\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n> 5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n    |     ^\n  6 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mFOO\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mfoo\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  7 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mBAR\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mbar\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name: "5th line, 5th column, padding=2, no indicators",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(5),
				WithColumn(5),
				WithPadding(2),
				WithNoIndicators(),
			},
			want: "  3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  4 | \x1b[1m\x1b[30m  \x1b[0m\x1b[33mdefault\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  6 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mFOO\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mfoo\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  7 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mBAR\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mbar\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name: "10th line, 1st column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(10),
				WithColumn(1),
			},
			want: "> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     | ^",
		},
		{
			name: "10th line, 23rd column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(10),
				WithColumn(23),
			},
			want: "> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     |                       ^",
		},
		{
			name: "10th line, 24th column (out of bounds)",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(10),
				WithColumn(24),
			},
			want: "> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name: "10th line, 23rd column, padding=2",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(10),
				WithColumn(23),
				WithPadding(2),
			},
			want: "   8 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mcmds\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   9 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.FOO}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n> 10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     |                       ^",
		},
		{
			name: "5th line, 5th column, padding=100",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(5),
				WithColumn(5),
				WithPadding(100),
			},
			want: "   1 | \x1b[33mversion\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36m3\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   2 | \x1b[1m\x1b[30m\x1b[0m\n   3 | \x1b[33mtasks\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   4 | \x1b[1m\x1b[30m  \x1b[0m\x1b[33mdefault\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n>  5 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mvars\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n     |     ^\n   6 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mFOO\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mfoo\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   7 | \x1b[1m\x1b[30m      \x1b[0m\x1b[33mBAR\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m \x1b[0m\x1b[36mbar\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   8 | \x1b[1m\x1b[30m    \x1b[0m\x1b[33mcmds\x1b[0m\x1b[1m\x1b[30m:\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n   9 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.FOO}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
		{
			name: "11th line (out of bounds), 1st column",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(11),
				WithColumn(1),
			},
			want: "",
		},
		{
			name: "11th line (out of bounds), 1st column, padding=2",
			b:    []byte(sample),
			opts: []SnippetOption{
				WithLine(11),
				WithColumn(1),
				WithPadding(2),
			},
			want: "   9 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.FOO}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m\n  10 | \x1b[1m\x1b[30m      \x1b[0m\x1b[1m\x1b[30m- \x1b[0m\x1b[36mecho \"{{.BAR}}\"\x1b[0m\x1b[1m\x1b[30m\x1b[0m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			snippet := NewSnippet(tt.b, tt.opts...)
			got := snippet.String()
			if strings.Contains(got, "\t") {
				t.Fatalf("tab character found in snippet - check the sample string")
			}
			require.Equal(t, tt.want, got)
		})
	}
}
