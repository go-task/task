package ast

type Prompt []string

func (p *Prompt) UnmarshalYAML(unmarshal func(any) error) error {
	var m []string
	err := unmarshal(&m)
	if err == nil {
		*p = m
		return nil
	}

	var s string
	err = unmarshal(&s)
	if err != nil {
		return err
	}

	*p = []string{s}
	return nil
}
