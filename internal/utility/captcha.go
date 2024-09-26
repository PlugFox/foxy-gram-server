package utility

/* import (
	"io"
	"time"

	captcha "github.com/dchest/captcha"
	"github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/model"
)



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
 */