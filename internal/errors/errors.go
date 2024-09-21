package errors

import (
	"errors"
	"fmt"
)

// Статическая ошибка для несоответствия типов.
var ErrorUnexpectedType = errors.New("unexpected type")

// Функция для оборачивания статической ошибки дополнительной информацией.
func WrapUnexpectedType(expected string, actual interface{}) error {
	return fmt.Errorf("%w: expected %s, got %T", ErrorUnexpectedType, expected, actual)
}
