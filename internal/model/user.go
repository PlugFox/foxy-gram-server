package model

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/gob"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm"
)

type (
	UserID int64
)

type User struct {
	ID UserID `gorm:"PrimaryKey" json:"id"` // Unique identifier for this user or bot.

	// User fields
	FirstName    string `json:"first_name"`    // User's or bot's first name.
	LastName     string `json:"last_name"`     // User's or bot's last name.
	Username     string `json:"username"`      // User's or bot's username.
	LanguageCode string `json:"language_code"` // IETF language tag of the user's language.
	IsPremium    bool   `json:"is_premium"`    // True, if the user is a premium user.
	IsBot        bool   `json:"is_bot"`        // True, if the user is a bot.

	// Meta fields
	LastSeen  sql.NullTime   `json:"last_seen"`                        // Unix time when the user was last seen.
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"` // Time when the user was last updated.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`          // Soft delete.
	Extra     string         `json:"extra"`                            // Extra data.
}

// Update seen time for the user
func (obj *User) Seen() *User {
	obj.LastSeen = sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}
	return obj
}

// TableName - set the table name
func (User) TableName() string {
	return "users"
}

// Hash - calculate the hash of the object
func (obj *User) Hash() ([32]byte, error) {
	hashable := make(map[string]interface{})

	// Используем рефлексию для извлечения значений полей с тегом "hash"
	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		_, ok := field.Tag.Lookup("hash") // Проверяем наличие тега "hash" без проверки значения
		if ok {                           // Если тег "hash" присутствует
			fieldValue := val.Field(i)
			hashable[field.Name] = fieldValue.Interface()
		}
	}

	// Сериализуем выбранные поля через gob
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(hashable)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to encode hashable fields: %w", err)
	}

	// Вычисляем sha256-хэш от сериализованных данных
	hash := sha256.Sum256(buf.Bytes())
	return hash, nil

	// Возвращаем хэш в виде строки
	// return fmt.Sprintf("%x", hash), nil
}
