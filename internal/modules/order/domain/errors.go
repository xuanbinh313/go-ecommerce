package domain

type InvalidDataError struct{ Message string }

func (e *InvalidDataError) Error() string { return e.Message }
func InvalidData(message string) error    { return &InvalidDataError{Message: message} }
