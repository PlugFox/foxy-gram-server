package model

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/utility"
	"gorm.io/gorm"
)

type (
	UserID int64
)

type User struct {
	ID UserID `gorm:"PrimaryKey" hash:"x" json:"id"` // Unique identifier for this user or bot.

	// User fields
	FirstName    string `hash:"x" json:"first_name"`    // User's or bot's first name.
	LastName     string `hash:"x" json:"last_name"`     // User's or bot's last name.
	Username     string `hash:"x" json:"username"`      // User's or bot's username.
	LanguageCode string `hash:"x" json:"language_code"` // IETF language tag of the user's language.
	IsPremium    bool   `hash:"x" json:"is_premium"`    // True, if the user is a premium user.
	IsBot        bool   `hash:"x" json:"is_bot"`        // True, if the user is a bot.

	// Additional fields
	LastSeen sql.NullTime `hash:"x" json:"last_seen"` // Unix time when the user was last seen.

	// Meta fields
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"` // Time when the user was last updated.
	DeletedAt gorm.DeletedAt `gorm:"index"          json:"deleted_at"` // Soft delete.
	Extra     string         `json:"extra"`                            // Extra data.
}

// Update seen time for the user.
func (obj *User) Seen() *User {
	obj.LastSeen = sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}

	return obj
}

// TableName - set the table name.
func (User) TableName() string {
	return "users"
}

// GetID - get the user ID.
func (obj *User) GetID() int64 {
	return int64(obj.ID)
}

// ToInt64 - get the user ID.
func (id UserID) ToInt64() int64 {
	return int64(id)
}

// ToString - get the user ID.
func (id UserID) ToString() string {
	return strconv.FormatInt(int64(id), 10)
}

// Hash - calculate the hash of the object.
func (obj *User) Hash() (string, error) {
	return utility.Hash(obj)
}
