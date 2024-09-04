package model

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type (
	UserID    int64
	ChannelID int64
	MessageID int64
)

type User struct {
	ID UserID `gorm:"PrimaryKey" json:"id"` // Unique identifier for this user or bot.

	// 	Username string `gorm:"uniqueIndex" json:"username"`      // User's or bot's username.

	// User fields
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`
	IsBot        bool   `json:"is_bot"`

	// Meta fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"` // Time when the user registered.
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"` // Time when the user was last updated.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`          // Soft delete.
	LastSeen  sql.NullTime   `json:"last_seen"`                        // Unix time when the user was last seen.
	Extra     string         `json:"extra"`                            // Extra data.
}

// Update seen time for the user
func (u *User) Seen() *User {
	u.LastSeen = sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}
	return u
}
