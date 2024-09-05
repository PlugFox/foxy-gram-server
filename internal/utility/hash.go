package utility

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"
	"sort"
)

var errorNoHashableFields = errors.New("no hashable fields found")

// Hash - calculate the hash of the object
func Hash(obj interface{}) (string /* [32]byte */, error) {
	hashable := make(map[string]interface{})

	// Получаем отражение объекта и его тип
	val := reflect.ValueOf(obj)

	// Если obj - это указатель, разыменуем его
	val = reflect.Indirect(val)
	typ := val.Type()

	// Используем рефлексию для извлечения значений полей с тегом "hash"
	hasFields := false
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		_, ok := field.Tag.Lookup("hash") // Проверяем наличие тега "hash" без проверки значения
		if ok {                           // Если тег "hash" присутствует
			fieldValue := val.Field(i)
			hashable[field.Name] = fieldValue.Interface()
			hasFields = true
		}
	}

	// Если не найдено полей с тегом "hash", возвращаем ошибку
	if !hasFields {
		return "", errorNoHashableFields
	}

	// Сортируем ключи карты hashable для последовательной сериализации
	keys := make([]string, 0, len(hashable))
	for k := range hashable {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Сортируем ключи по алфавиту

	// Сериализуем выбранные поля через gob
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	// Кодируем поля в отсортированном порядке
	for _, key := range keys {
		err := enc.Encode(hashable[key])
		if err != nil {
			return "", fmt.Errorf("failed to encode hashable fields: %w", err)
		}
	}

	// Вычисляем sha256-хэш от сериализованных данных
	hash := sha256.Sum256(buf.Bytes())

	// Возвращаем хэш в виде строки
	return fmt.Sprintf("%x", hash), nil
}
