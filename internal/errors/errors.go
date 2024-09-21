package errors

import (
	"errors"
	"fmt"
)

// Static error for unexpected type.
var ErrorUnexpectedType = errors.New("unexpected type")

// WrapUnexpectedType wraps the error for unexpected type.
func WrapUnexpectedType(expected string, actual interface{}) error {
	return fmt.Errorf("%w: expected %s, got %T", ErrorUnexpectedType, expected, actual)
}
