package model

import (
	"strconv"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/utility"
	"gorm.io/gorm"
)

type (
	ChatID int64
)

type Chat struct {
	ID        ChatID `hash:"x" gorm:"PrimaryKey" json:"id"` // Unique identifier for the chat.
	Type      string `hash:"x" json:"type"`                 // Chat type (e.g., "private", "group", "supergroup", "channel").
	Title     string `hash:"x" json:"title"`                // Chat title for groups, supergroups, and channels.
	Username  string `hash:"x" json:"username"`             // Chat's username (optional).
	IsPrivate bool   `hash:"x" json:"is_private"`           // Whether this chat is a private conversation.

	// Meta fields
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"` // Time when the chat was last updated.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`          // Soft delete.
	Extra     string         `json:"extra"`                            // Extra data.
}

// TableName - set the table name
func (Chat) TableName() string {
	return "chats"
}

// GetID - get the chat ID
func (obj *Chat) GetID() int64 {
	return int64(obj.ID)
}

// ToInt64 - get the chat ID
func (id ChatID) ToInt64() int64 {
	return int64(id)
}

// ToString - get the chat ID
func (id ChatID) ToString() string {
	return strconv.FormatInt(int64(id), 10)
}

// Hash - calculate the hash of the object
func (obj *Chat) Hash() (string, error) {
	return utility.Hash(obj)
}
