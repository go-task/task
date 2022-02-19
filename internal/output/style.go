package output

import (
	"fmt"
)

// Style of the Task output
type Style struct {
	// Name of the Style.
	Name       string `yaml:"-"`
	// Group specific style
	Group GroupStyle
}

// Build the Output for the requested Style.
func (s *Style) Build() (Output, error) {
	switch s.Name {
	case "interleaved", "":
		return Interleaved{}, s.ensureGroupStyleUnset()
	case "group":
		return Group{
			Begin: s.Group.Begin,
			End:   s.Group.End,
		}, nil
	case "prefixed":
		return Prefixed{}, s.ensureGroupStyleUnset()
	default:
		return nil, fmt.Errorf(`task: output style %q not recognized`, s.Name)
	}
}

func (s *Style) ensureGroupStyleUnset() error {
	if s.Group.IsSet() {
		return fmt.Errorf("task: output style %q does not support the group begin/end parameter", s.Name)
	}
	return nil
}

// IsSet returns true if and only if a custom output style is set.
func (s *Style) IsSet() bool {
	return s.Name != ""
}

// UnmarshalYAML implements yaml.Unmarshaler
// It accepts a scalar node representing the Style.Name or a mapping node representing the GroupStyle.
func (s *Style) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err == nil {
		return s.UnmarshalText([]byte(name))
	}
	var tmp struct {
		Group *GroupStyle
	}
	if err := unmarshal(&tmp); err != nil {
		return fmt.Errorf("task: output style must be a string or mapping with a \"group\" key: %w", err)
	}
	if tmp.Group == nil {
		return fmt.Errorf("task: output style must have the \"group\" key when in mapping form")
	}
	*s = Style{
		Name: "group",
		Group: *tmp.Group,
	}
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler
// It accepts the Style.Node
func (s *Style) UnmarshalText(text []byte) error {
	tmp := Style{Name: string(text)}
	if _, err := tmp.Build(); err != nil {
		return err
	}
	return nil
}

// GroupStyle is the style options specific to the Group style.
type GroupStyle struct{
	Begin, End string
}

// IsSet returns true if and only if a custom output style is set.
func (g *GroupStyle) IsSet() bool {
	return g != nil && *g != GroupStyle{}
}
