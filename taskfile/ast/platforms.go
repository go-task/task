package ast

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/goext"
)

// Platform represents GOOS and GOARCH values
type Platform struct {
	OS   string
	Arch string
}

func (p *Platform) DeepCopy() *Platform {
	if p == nil {
		return nil
	}
	return &Platform{
		OS:   p.OS,
		Arch: p.Arch,
	}
}

type ErrInvalidPlatform struct {
	Platform string
}

func (err *ErrInvalidPlatform) Error() string {
	return fmt.Sprintf(`invalid platform "%s"`, err.Platform)
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (p *Platform) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var platform string
		if err := node.Decode(&platform); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		if err := p.parsePlatform(platform); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		return nil
	}
	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("platform")
}

// parsePlatform takes a string representing an OS/Arch combination (or either on their own)
// and parses it into the Platform struct. It returns an error if the input string is invalid.
// Valid combinations for input: OS, Arch, OS/Arch
func (p *Platform) parsePlatform(input string) error {
	splitValues := strings.Split(input, "/")
	if len(splitValues) > 2 {
		return &ErrInvalidPlatform{Platform: input}
	}
	if err := p.parseOsOrArch(splitValues[0]); err != nil {
		return &ErrInvalidPlatform{Platform: input}
	}
	if len(splitValues) == 2 {
		if err := p.parseArch(splitValues[1]); err != nil {
			return &ErrInvalidPlatform{Platform: input}
		}
	}
	return nil
}

// parseOsOrArch will check if the given input is a valid OS or Arch value.
// If so, it will store it. If not, an error is returned
func (p *Platform) parseOsOrArch(osOrArch string) error {
	if osOrArch == "" {
		return fmt.Errorf("task: Blank OS/Arch value provided")
	}
	if goext.IsKnownOS(osOrArch) {
		p.OS = osOrArch
		return nil
	}
	if goext.IsKnownArch(osOrArch) {
		p.Arch = osOrArch
		return nil
	}
	return fmt.Errorf("task: Invalid OS/Arch value provided (%s)", osOrArch)
}

func (p *Platform) parseArch(arch string) error {
	if arch == "" {
		return fmt.Errorf("task: Blank Arch value provided")
	}
	if p.Arch != "" {
		return fmt.Errorf("task: Multiple Arch values provided")
	}
	if goext.IsKnownArch(arch) {
		p.Arch = arch
		return nil
	}
	return fmt.Errorf("task: Invalid Arch value provided (%s)", arch)
}
