package utility

import (
	"bytes"

	captcha "github.com/dchest/captcha"
	"github.com/plugfox/foxy-gram-server/internal/config"
)

func GenerateCaptcha(config config.CaptchaConfig) (*bytes.Buffer, error) {
	id := captcha.NewLen(config.Length)
	w := new(bytes.Buffer)
	err := captcha.WriteImage(w, id, config.Width, config.Height)
	if err != nil {
		return w, err
	}
	return w, nil
}
