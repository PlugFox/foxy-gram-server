package utility

import (
	"io"
	"time"

	captcha "github.com/dchest/captcha"
	"github.com/plugfox/foxy-gram-server/internal/config"
)

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
	_, err := captcha.NewImage(string(captcha.RandomDigits(20)), digits, config.Width, config.Height).WriteTo(writer)
	if err != nil {
		return nil, err
	}
	return &Captcha{
		Digits:     digits,
		Length:     config.Length,
		Width:      config.Width,
		Height:     config.Height,
		Expiration: captcha.Expiration,
		ExpiresAt:  time.Now().Add(captcha.Expiration),
	}, nil
}

// VerifyCaptcha verifies the captcha with the given id and digits.
func VerifyCaptcha(id string, digits []byte) bool {
	return captcha.VerifyString(id, string(digits))
}

// Refresh renews the captcha image and expiration time.
func (c *Captcha) Refresh(writer io.Writer) error {
	c.Digits = captcha.RandomDigits(c.Length)
	_, err := captcha.NewImage(string(captcha.RandomDigits(20)), c.Digits, c.Width, c.Height).WriteTo(writer)
	if err != nil {
		return err
	}
	c.Expiration = captcha.Expiration
	c.ExpiresAt = time.Now().Add(c.Expiration)
	return nil
}
