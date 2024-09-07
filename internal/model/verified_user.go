package model

import "time"

type VerifiedUser struct {
	ID             uint      `gorm:"primaryKey"`
	Email          string    `gorm:"unique;not null"`
	VerifiedAt     time.Time `gorm:"not null"`
	VerificationID string    `gorm:"unique;not null"`
}
