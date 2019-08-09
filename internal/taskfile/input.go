package taskfile

import (
	"fmt"
	"regexp"
	"strings"
)

// Inputs represents a group of Input
type Inputs map[string]*Input

// Input represents an interactive input variable
type Input struct {
	Desc      string
	Required  bool
	Default   string
	Validator string
}

// FullTitle returns the input full title suffixed
func (i *Input) FullTitle(name string) (t string) {
	t = i.Desc
	if t == "" {
		t = name
	}

	st := i.subtitle()
	if st == "" {
		return fmt.Sprintf("%s:", t)
	}

	return fmt.Sprintf("%s (%s):", t, st)
}

// subtitle returns the input subtitle infos
func (i *Input) subtitle() (subtitle string) {
	infos := []string{}

	// Required input
	if i.Required {
		infos = append(infos, "required: yes")
	}
	// Default value
	if i.Default != "" {
		infos = append(infos, fmt.Sprintf("default: \"%s\"", i.Default))
	}
	// Validator value
	if i.Validator != "" {
		infos = append(infos, fmt.Sprintf("validator: \"%s\"", i.Validator))
	}

	if len(infos) > 0 {
		subtitle = fmt.Sprintf("%s", strings.Join(infos, ","))
	}

	return
}

// Validate returns if the input is validate
func (i *Input) Validate(val string) bool {
	if i.Validator == "" {
		return true
	}

	m, _ := regexp.MatchString(i.Validator, val)
	return m
}
