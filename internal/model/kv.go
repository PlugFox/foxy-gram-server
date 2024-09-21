package model

import (
	"bytes"
	"encoding/gob"
	"errors"
	"time"
)

var errorValueEmpty = errors.New("value is empty")

// KeyValue - key-value pair
// Save value as a byte array to support any type of value with gob
type KeyValue struct {
	// Key-value fields
	Key   string `hash:"x" gorm:"primaryKey"`
	Value []byte `hash:"x" json:"value"` // Save the value as a byte array.

	// Meta fields
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Time when the user was last updated.
	Extra     string    `json:"extra"`                            // Extra data for the key-value pair.
}

// TableName - set the table name
func (KeyValue) TableName() string {
	return "kv"
}

// Set the value to the key-value pair
func (kv *KeyValue) SetValue(value interface{}) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(value)
	if err != nil {
		return err
	}
	kv.Value = buffer.Bytes()
	return nil
}

// Get the value from the key-value pair
func (kv *KeyValue) GetValue(out interface{}) error {
	if len(kv.Value) == 0 {
		return errorValueEmpty
	}
	buffer := bytes.NewBuffer(kv.Value)
	dec := gob.NewDecoder(buffer)
	return dec.Decode(out)
}
