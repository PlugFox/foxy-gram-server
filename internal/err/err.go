package err

import (
	"errors"
	"fmt"
)

var (
	ErrorUnexpectedType                = errors.New("unexpected type")                  // Static error for unexpected type.
	ErrorGlobalVariablesNotInitialized = errors.New("global variables not initialized") // Static error for global variables not initialized.
)

// WrapUnexpectedType wraps the error for unexpected type.
func WrapUnexpectedType(expected string, actual interface{}) error {
	return fmt.Errorf("%w: expected %s, got %T", ErrorUnexpectedType, expected, actual)
}
