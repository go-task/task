package taskfile

import (
	"fmt"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Platform represents GOOS and GOARCH values
type Platform struct {
	OS   string
	Arch string
}

// ParsePlatform takes a string representing an OS/Arch combination (or either on their own)
// and parses it into the Platform struct. It returns an error if the input string is invalid.
// Valid combinations for input: OS, Arch, OS/Arch
func (p *Platform) ParsePlatform(input string) error {
	// tidy up input
	platformString := strings.ToLower(strings.TrimSpace(input))
	splitValues := strings.Split(platformString, "/")
	if len(splitValues) > 2 {
		return fmt.Errorf("task: Invalid OS/Arch provided: %s", input)
	}
	err := p.parseOsOrArch(splitValues[0])
	if err != nil {
		return err
	}
	if len(splitValues) == 2 {
		return p.parseArch(splitValues[1])
	}
	return nil
}

// supportedOSes is a list of supported OSes
var supportedOSes = map[string]struct{}{
	"windows": {},
	"darwin":  {},
	"linux":   {},
	"freebsd": {},
}

func isSupportedOS(input string) bool {
	_, exists := supportedOSes[input]
	return exists
}

// supportedArchs is a list of supported architectures
var supportedArchs = map[string]struct{}{
	"amd64": {},
	"arm64": {},
	"386":   {},
}

func isSupportedArch(input string) bool {
	_, exists := supportedArchs[input]
	return exists
}

// MatchesCurrentPlatform returns true if the platform matches the current platform
func (p *Platform) MatchesCurrentPlatform() bool {
	return (p.OS == "" || p.OS == runtime.GOOS) &&
		(p.Arch == "" || p.Arch == runtime.GOARCH)
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (p *Platform) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var platform string
		if err := node.Decode(&platform); err != nil {
			return err
		}
		if err := p.ParsePlatform(platform); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into platform", node.Line, node.ShortTag())
}

// parseOsOrArch will check if the given input is a valid OS or Arch value.
// If so, it will store it. If not, an error is returned
func (p *Platform) parseOsOrArch(osOrArch string) error {
	if osOrArch == "" {
		return fmt.Errorf("task: Blank OS/Arch value provided")
	}
	if isSupportedOS(osOrArch) {
		p.OS = osOrArch
		return nil
	}
	if isSupportedArch(osOrArch) {
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
	if isSupportedArch(arch) {
		p.Arch = arch
		return nil
	}
	return fmt.Errorf("task: Invalid Arch value provided (%s)", arch)
}
