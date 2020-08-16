package status

var (
	_ Checker = &Timestamp{}
	_ Checker = &Checksum{}
	_ Checker = None{}
)

// Checker is an interface that checks if the status is up-to-date
type Checker interface {
	IsUpToDate() (bool, error)
	Value() (interface{}, error)
	OnError() error
	Kind() string
}
