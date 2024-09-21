package model

import (
	"time"

	"github.com/plugfox/foxy-gram-server/internal/utility"
)

// Verified user represents a verified user in the system
type VerifiedUser struct {
	ID         UserID    `hash:"x" gorm:"primaryKey" json:"id"`
	VerifiedAt time.Time `hash:"x" gorm:"not null" json:"verified_at"` // The time when the user was verified
	Reason     string    `hash:"x" gorm:"not null" json:"reason"`      // Reason for the verification
	// ExpiresAt  sql.NullTime `hash:"x" gorm:"null" json:"expires_at"`      // Expiry time of the verification, null if indefinite

	// Meta fields
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Time when the user was last updated.
	Extra     string    `json:"extra"`                            // Extra data.
}

// TableName - set the table name
func (VerifiedUser) TableName() string {
	return "verified"
}

// GetID - get the user ID
func (obj *VerifiedUser) GetID() int64 {
	return int64(obj.ID)
}

// Hash - calculate the hash of the object
func (obj *VerifiedUser) Hash() (string, error) {
	return utility.Hash(obj)
}
