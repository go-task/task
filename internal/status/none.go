package status

// None is a no-op Checker
type None struct{}

// IsUpToDate implements the Checker interface
func (None) IsUpToDate() (bool, error) {
	return false, nil
}

// Value implements the Checker interface
func (None) Value() (interface{}, error) {
	return "", nil
}

func (None) Kind() string {
	return "none"
}

// OnError implements the Checker interface
func (None) OnError() error {
	return nil
}
