package model

import (
	"database/sql"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/utility"
)

// BannedUser represents a banned user in the system
type BannedUser struct {
	ID        UserID       `hash:"x" gorm:"primaryKey" json:"id"`
	BannedAt  time.Time    `hash:"x" gorm:"not null" json:"banned_at"` // The time when the user was banned
	Reason    string       `hash:"x" gorm:"not null" json:"reason"`    // Reason for the ban
	ExpiresAt sql.NullTime `hash:"x" gorm:"null" json:"expires_at"`    // Expiry time of the ban, null if indefinite

	// Meta fields
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Time when the user was last updated
	Extra     string    `json:"extra"`                            // Extra data
}

// TableName - set the table name
func (BannedUser) TableName() string {
	return "banned"
}

// GetID - get the user ID
func (c *BannedUser) GetID() int64 {
	return int64(c.ID)
}

// Hash - calculate the hash of the object
func (obj *BannedUser) Hash() (string, error) {
	return utility.Hash(obj)
}
