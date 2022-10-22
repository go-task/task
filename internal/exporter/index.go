package exporter

import "fmt"

type Type string

const (
	GithubActions Type = "github_actions"
)

// AllowedExporterTypeEnumValues lists the allowed values of GithubActions enum
var AllowedExporterTypeEnumValues = []Type{
	GithubActions,
}

// UnmarshalTypes returns a pointer to a valid []exporter.Type slice,
// for the value passed as argument, or an error if any value passed is not allowed by the enum
func UnmarshalTypes(values []string) (*[]Type, error) {
	var result []Type

	for _, value := range values {
		item, err := unmarshalType(value)
		if err != nil {
			return nil, err
		}
		result = append(result, *item)
	}

	return &result, nil
}

// unmarshalType returns a pointer to a valid exporter.Type
// for the value passed as argument, or an error if the value passed is not allowed by the enum
func unmarshalType(v string) (*Type, error) {
	ev := Type(v)
	if ev.IsValid() {
		return &ev, nil
	} else {
		return nil, fmt.Errorf("invalid value '%v' for exporter.Type: valid values are %v", v, AllowedExporterTypeEnumValues)
	}
}

// IsValid return true if the value is valid for the enum, false otherwise
func (v Type) IsValid() bool {
	for _, existing := range AllowedExporterTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}
