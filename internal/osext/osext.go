package osext

import (
	"os"

	"github.com/mitchellh/go-homedir"
)

// Expand is an improved version of os.ExpandEnv,
// that not only expand enrionment variable ($GOPATH/src/github.com/...)
// but also expands "~" as the home directory.
func Expand(s string) (string, error) {
	s = os.ExpandEnv(s)

	var err error
	s, err = homedir.Expand(s)
	if err != nil {
		return "", err
	}

	return s, nil
}
