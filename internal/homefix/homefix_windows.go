package homefix

import (
	"os"

	"github.com/mitchellh/go-homedir"
)

func init() {
	if os.Getenv("HOME") == "" {
		if home, err := homedir.Dir(); err == nil {
			os.Setenv("HOME", home)
		}
	}
}
