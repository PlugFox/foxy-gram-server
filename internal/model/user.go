package model

import (
	"database/sql"
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
func (u *User) Seen() *User {
	u.LastSeen = sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}
	return u
}

// TableName - set the table name
func (User) TableName() string {
	return "users"
}
