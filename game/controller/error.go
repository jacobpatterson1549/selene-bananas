package controller

// gameWarning is an error that represents a user error.
type gameWarning string

// Error returns the string of the error.
func (w gameWarning) Error() string {
	return string(w)
}
