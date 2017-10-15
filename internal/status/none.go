package status

// None is a no-op Checker
type None struct{}

// IsUpToDate implements the Checker interface
func (None) IsUpToDate() (bool, error) {
	return false, nil
}

// OnError implements the Checker interface
func (None) OnError() error {
	return nil
}
