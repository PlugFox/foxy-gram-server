package model

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/plugfox/foxy-gram-server/internal/global"
	"github.com/plugfox/foxy-gram-server/internal/utility"
)

// idLength is the length of the captcha id to be used in generators.
const idLength = 20

// Captcha - represents a captcha with an image and expiration time.
type Captcha struct {
	ID int64 `gorm:"PrimaryKey" hash:"x" json:"id"` // Captcha ID.

	UserID int64 `gorm:"index" hash:"x" json:"user_id"` // Identifier for the user.

	ChatID int64 `gorm:"index" hash:"x" json:"chat_id"` // Identifier for the chat.

	MessageID int64 `gorm:"index" hash:"x" json:"message_id"` // Identifier for the message.

	Digits string `hash:"x" json:"digits"` // Digits of the captcha.

	Length int `hash:"x" json:"length"` // Length of the captcha.

	Width int `hash:"x" json:"width"` // Width of the captcha.

	Height int `hash:"x" json:"height"` // Height of the captcha.

	Expiration time.Duration `hash:"x" json:"expiration"` // Expiration time of the captcha.

	ExpiresAt time.Time `hash:"x" json:"expires_at"` // Time when the captcha expires.

	// Meta fields
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Time when the captcha was last updated.
}

// TableName - set the table name.
func (Captcha) TableName() string {
	return "captchas"
}

// GetID - get the captcha ID.
func (obj *Captcha) GetID() int64 {
	return obj.ID
}

// Hash - calculate the hash of the object.
func (obj *Captcha) Hash() (string, error) {
	return utility.Hash(obj)
}

// Expired - checks if the captcha has expired.
func (obj *Captcha) Expired() bool {
	return obj.ExpiresAt.Before(time.Now())
}

// Generates a new captcha with the given configuration.
func GenerateCaptcha(writer io.Writer) (*Captcha, error) {
	config := global.Config.Captcha
	randomDigits := captcha.RandomDigits(config.Length)

	id := string(captcha.RandomDigits(idLength))
	image := captcha.NewImage(id, randomDigits, config.Width, config.Height)
	if _, err := image.WriteTo(writer); err != nil {
		return nil, err
	}

	var strNumbers []string
	for _, b := range randomDigits {
		strNumbers = append(strNumbers, strconv.Itoa(int(b)))
	}

	digits := strings.Join(strNumbers, "")

	return &Captcha{
		Digits:     digits,
		Length:     config.Length,
		Width:      config.Width,
		Height:     config.Height,
		Expiration: config.Expiration,
		ExpiresAt:  time.Now().Add(config.Expiration),
	}, nil
}
