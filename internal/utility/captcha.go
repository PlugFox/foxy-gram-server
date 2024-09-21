package utility

import (
	"io"
	"time"

	captcha "github.com/dchest/captcha"
	"github.com/plugfox/foxy-gram-server/internal/config"
)

// idLength is the length of the captcha id to be used in generators.
const idLength = 20

// Captcha represents a captcha with an image and expiration time.
type Captcha struct {
	Digits     []byte
	Length     int
	Width      int
	Height     int
	Expiration time.Duration
	ExpiresAt  time.Time
}

// GenerateCaptcha generates a new captcha with the given configuration.
func GenerateCaptcha(config config.CaptchaConfig, writer io.Writer) (*Captcha, error) {
	digits := captcha.RandomDigits(config.Length)

	id := string(captcha.RandomDigits(idLength))
	if _, err := captcha.NewImage(id, digits, config.Width, config.Height).WriteTo(writer); err != nil {
		return nil, err
	}

	return &Captcha{
		Digits:     digits,
		Length:     config.Length,
		Width:      config.Width,
		Height:     config.Height,
		Expiration: config.Expiration,
		ExpiresAt:  time.Now().Add(config.Expiration),
	}, nil
}

// VerifyCaptcha verifies the captcha with the given id and digits.
func VerifyCaptcha(id string, digits []byte) bool {
	return captcha.VerifyString(id, string(digits))
}

// Refresh renews the captcha image and expiration time.
func (c *Captcha) Refresh(writer io.Writer) error {
	c.Digits = captcha.RandomDigits(c.Length)

	id := string(captcha.RandomDigits(idLength))
	if _, err := captcha.NewImage(id, c.Digits, c.Width, c.Height).WriteTo(writer); err != nil {
		return err
	}

	c.ExpiresAt = time.Now().Add(c.Expiration)

	return nil
}
