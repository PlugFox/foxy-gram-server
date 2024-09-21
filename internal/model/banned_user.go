package model

import (
	"database/sql"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/utility"
)

// BannedUser represents a banned user in the system.
type BannedUser struct {
	ID        UserID       `gorm:"primaryKey" hash:"x" json:"id"`
	BannedAt  time.Time    `gorm:"not null"   hash:"x" json:"banned_at"`  // The time when the user was banned
	Reason    string       `gorm:"not null"   hash:"x" json:"reason"`     // Reason for the ban
	ExpiresAt sql.NullTime `gorm:"null"       hash:"x" json:"expires_at"` // Expiry time of the ban, null if indefinite

	// Meta fields
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Time when the user was last updated
	Extra     string    `json:"extra"`                            // Extra data
}

// TableName - set the table name.
func (BannedUser) TableName() string {
	return "banned"
}

// GetID - get the user ID.
func (obj *BannedUser) GetID() int64 {
	return int64(obj.ID)
}

// Hash - calculate the hash of the object.
func (obj *BannedUser) Hash() (string, error) {
	return utility.Hash(obj)
}
