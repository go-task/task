package term

import (
	"os"

	"golang.org/x/term"
)

func IsTerminal() bool {
	fd := os.Stdin.Fd()
	return term.IsTerminal(int(fd)) && term.IsTerminal(int(os.Stdout.Fd())) //nolint:gosec
}
