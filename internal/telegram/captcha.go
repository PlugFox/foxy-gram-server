package telegram

import (
	"bytes"
	"fmt"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/utility"
	tele "gopkg.in/telebot.v3"
)

//nolint:unused
func buildCaptchaMessage(conf config.CaptchaConfig, user tele.User) (*captchaMessage, error) {
	var caption string
	if username := user.Username; username != "" {
		caption = fmt.Sprintf("@%s, please solve the captcha.\nReply with the code in the image.", username)
	} else if firstName := user.FirstName; firstName != "" {
		caption = "%s, please solve the captcha.\nReply with the code in the image."
	} else {
		caption = "Please solve the captcha.\nReply with the code in the image."
	}

	buffer := new(bytes.Buffer)

	captcha, err := utility.GenerateCaptcha(conf, buffer)
	if err != nil {
		return nil, err
	}

	refreshBtn := tele.InlineButton{Text: "Refresh 🔄", Unique: "refresh_captcha"}
	cancelBtn := tele.InlineButton{Text: "Cancel ❌", Unique: "cancel_captcha"}

	return &captchaMessage{
		captcha: captcha,
		photo: tele.Photo{
			File:    tele.FromReader(buffer),
			Width:   captcha.Width,
			Height:  captcha.Height,
			Caption: caption,
		},
		reply: tele.ReplyMarkup{
			ForceReply: true,
			Selective:  user.Username != "",
			InlineKeyboard: [][]tele.InlineButton{
				{},
				{},
				{},
				{},
				{cancelBtn, refreshBtn},
			},
		},
	}, nil
}

// todo: implement sendCaptchaMessage
/* func sendCaptchaMessage(conf config.CaptchaConfig, bot *tele.Bot, chat *tele.Chat, user *tele.User) error {
	return nil
} */
