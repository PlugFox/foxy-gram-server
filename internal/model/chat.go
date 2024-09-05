package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm"
)

type (
	ChatID int64
)

type Chat struct {
	ID        ChatID `hash:"" gorm:"PrimaryKey" json:"id"` // Unique identifier for the chat.
	Type      string `hash:"" json:"type"`                 // Chat type (e.g., "private", "group", "supergroup", "channel").
	Title     string `hash:"" json:"title"`                // Chat title for groups, supergroups, and channels.
	Username  string `hash:"" json:"username"`             // Chat's username (optional).
	IsPrivate bool   `hash:"" json:"is_private"`           // Whether this chat is a private conversation.

	// Meta fields
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"` // Time when the chat was last updated.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`          // Soft delete.
	Extra     string         `json:"extra"`                            // Extra data.
}

// TableName - set the table name
func (Chat) TableName() string {
	return "chats"
}

// Hash - calculate the hash of the object
func (obj *Chat) Hash() ([32]byte, error) {
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
