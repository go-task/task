package execext

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func Build(cmd string, args ...string) (string, error) {
	quotedArgs := make([]string, 0, len(args))

	for _, arg := range args {
		quoted, err := syntax.Quote(arg, syntax.LangBash)
		if err != nil {
			return "", err
		}
		quotedArgs = append(quotedArgs, quoted)
	}

	return fmt.Sprintf("%s %s", cmd, strings.Join(quotedArgs, " ")), nil
}
