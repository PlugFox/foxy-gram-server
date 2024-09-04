package model

import (
	"time"

	"gorm.io/gorm"
)

type (
	ChatID int64
)

type Chat struct {
	ID        ChatID `gorm:"PrimaryKey" json:"id"` // Unique identifier for the chat.
	Type      string `json:"type"`                 // Chat type (e.g., "private", "group", "supergroup", "channel").
	Title     string `json:"title"`                // Chat title for groups, supergroups, and channels.
	Username  string `json:"username"`             // Chat's username (optional).
	IsPrivate bool   `json:"is_private"`           // Whether this chat is a private conversation.

	// Meta fields
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"` // Time when the chat was last updated.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`          // Soft delete.
	Extra     string         `json:"extra"`                            // Extra data.
}

// TableName - set the table name
func (Chat) TableName() string {
	return "chats"
}
