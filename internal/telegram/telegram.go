// Library repository: https://github.com/tucnak/telebot

package telegram

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/converters"
	"github.com/plugfox/foxy-gram-server/internal/global"
	log "github.com/plugfox/foxy-gram-server/internal/log"
	"github.com/plugfox/foxy-gram-server/internal/model"
	"github.com/plugfox/foxy-gram-server/internal/storage"

	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	mw "gopkg.in/telebot.v3/middleware"
)

type Telegram struct {
	bot *tele.Bot
}

//nolint:funlen,gocognit,gocyclo,cyclop
func New(db *storage.Storage, httpClient *http.Client) (*Telegram, error) {
	pref := tele.Settings{
		Token:  global.Config.Telegram.Token,
		Client: httpClient,
		Poller: &tele.LongPoller{
			Timeout: global.Config.Telegram.Timeout,
		},
		OnError: func(err error, ctx tele.Context) {
			global.Logger.Error("telegram error", slog.String("error", err.Error()), slog.String("context", ctx.Text()))
		},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	// Global-scoped middleware:
	bot.Use(mw.Recover())
	bot.Use(mw.AutoRespond())
	bot.Use(mw.Logger(log.NewLogAdapter(global.Logger)))

	if global.Config.Telegram.IgnoreVia {
		bot.Use(mw.IgnoreVia())
	}

	bot.Use(verifyUserMiddleware(db, func(err error) {
		global.Logger.Error("verify user error", slog.String("error", err.Error()))
	}))

	bot.Use(verifyUserWithLocalDB(db, func(err error) {
		global.Logger.Error("verify user with local db error", slog.String("error", err.Error()))
	}))

	/* bot.Use(verifyUserWithCAS(db, httpClient, func(err error) {
		global.Logger.Error("verify user with cas error", slog.String("error", err.Error()))
	})) */

	bot.Use(verifyUserWithCaptcha(db, func(err error) {
		global.Logger.Error("verify user with captcha error", slog.String("error", err.Error()))
	}))

	if len(global.Config.Telegram.Whitelist) > 0 {
		bot.Use(mw.Whitelist(global.Config.Telegram.Whitelist...))
	}

	if len(global.Config.Telegram.Blacklist) > 0 {
		bot.Use(mw.Blacklist(global.Config.Telegram.Blacklist...))
	}

	// Store messages in the database
	bot.Use(storeMessagesMiddleware(db, func(err error) {
		global.Logger.Error("store message error", slog.String("error", err.Error()))
	}))

	// Group-scoped middleware:
	if len(global.Config.Telegram.Admins) > 0 {
		adminOnly := bot.Group()
		/* adminOnly.Handle("/ban", onBan)
		adminOnly.Handle("/kick", onKick) */
		adminOnly.Use(middleware.Whitelist(global.Config.Telegram.Admins...))
	}

	handlers := []interface{}{
		tele.OnText,
		tele.OnEdited,
		tele.OnUserJoined,
		tele.OnAddedToGroup,
		tele.OnPhoto,
		tele.OnAudio,
		tele.OnAnimation,
		tele.OnDocument,
		tele.OnSticker,
		tele.OnVideo,
		tele.OnVoice,
		tele.OnVideoNote,
		tele.OnContact,
		tele.OnLocation,
		tele.OnVenue,
		tele.OnPoll,
		tele.OnDice,
		tele.OnChannelPost,
		tele.OnMedia,
	}

	for _, handler := range handlers {
		bot.Handle(handler, func(_ tele.Context) error {
			return nil
		})
	}

	// Handle the captcha keyboard
	bot.Handle(&tele.Btn{Unique: captchaKeyboardUnique}, func(c tele.Context) error {
		user := c.Sender() // Get the user who clicked the button
		data := c.Data()   // Get the data from the button

		// Get the current captcha for the user
		captcha, err := db.GetCaptchaForUserID(user.ID)
		//nolint:gocritic
		if err != nil {
			return err
		} else if captcha == nil {
			return nil
		} else if captcha.Expired() {
			if err := db.DeleteCaptchaByID(captcha.ID); err != nil {
				return err
			}

			if err := c.Delete(); err != nil {
				global.Logger.Error("telegram: deleting captcha message error", slog.String("error", err.Error()))
			}

			return nil
		} else if captcha.MessageID != int64(c.Message().ID) {
			return nil
		}

		captcha.Expiration = global.Config.Captcha.Expiration
		editCaption := false

		switch data {
		case "captcha-refresh":
			// Refresh the captcha
			buffer := new(bytes.Buffer)

			defer buffer.Reset()

			if err := captcha.Refresh(buffer); err != nil {
				return err
			}

			// Create the photo message
			photo := tele.Photo{
				File:    tele.FromReader(buffer),
				Width:   captcha.Width,
				Height:  captcha.Height,
				Caption: captcha.Caption(user.Username, user.FirstName, user.LastName),
			}

			// Edit the existing message with the new photo
			if err := c.Edit(&photo, &tele.SendOptions{
				ReplyMarkup: &tele.ReplyMarkup{
					ForceReply:     false,
					Selective:      user.Username != "",
					InlineKeyboard: captchaKeyboardDefault().keyboard,
				},
			}); err != nil {
				return err
			}

			defer global.Metrics.LogChatEvent("captcha_refreshed", captcha.ChatID, map[string]interface{}{
				"chat_id": captcha.ChatID,
				"user_id": captcha.UserID,
			})

		case "captcha-backspace":
			// Backspace the last number in the captcha code
			if len(captcha.Input) > 0 {
				captcha.Input = captcha.Input[:len(captcha.Input)-1]
				editCaption = true
			}
		case "captcha-zero":
			// Add the number to the captcha code
			captcha.Input += "0"
			editCaption = true
		case "captcha-one":
			// Add the number to the captcha code
			captcha.Input += "1"
			editCaption = true
		case "captcha-two":
			// Add the number to the captcha code
			captcha.Input += "2"
			editCaption = true
		case "captcha-three":
			// Add the number to the captcha code
			captcha.Input += "3"
			editCaption = true
		case "captcha-four":
			// Add the number to the captcha code
			captcha.Input += "4"
			editCaption = true
		case "captcha-five":
			// Add the number to the captcha code
			captcha.Input += "5"
			editCaption = true
		case "captcha-six":
			// Add the number to the captcha code
			captcha.Input += "6"
			editCaption = true
		case "captcha-seven":
			// Add the number to the captcha code
			captcha.Input += "7"
			editCaption = true
		case "captcha-eight":
			// Add the number to the captcha code
			captcha.Input += "8"
			editCaption = true
		case "captcha-nine":
			// Add the number to the captcha code
			captcha.Input += "9"
			editCaption = true
		}

		// Check if the captcha code is correct
		if captcha.Validate() {
			if err := db.VerifyUser(&model.VerifiedUser{
				ID:         model.UserID(captcha.UserID),
				VerifiedAt: time.Now(),
				Reason:     "Captcha was solved",
			}); err != nil {
				global.Logger.Warn("Failed to verify user at database", slog.String("error", err.Error()))
			}

			if err := c.RespondText("You have been verified!"); err != nil {
				global.Logger.Warn("Failed to respond to the user", slog.String("error", err.Error()))
			}

			if err := c.Bot().Delete(c.Message()); err != nil {
				global.Logger.Warn("Failed to delete the message", slog.String("error", err.Error()))
			}

			if err := db.DeleteCaptchaByID(captcha.ID); err != nil {
				global.Logger.Warn("Failed to delete the captcha", slog.String("error", err.Error()))
			}

			defer global.Metrics.LogChatEvent("captcha_solved", captcha.ChatID, map[string]interface{}{
				"chat_id": captcha.ChatID,
				"user_id": captcha.UserID,
			})

			return nil
		} else if len(captcha.Input) >= len(captcha.Digits) {
			if err := c.RespondText("Invalid captcha code. Please try again."); err != nil {
				global.Logger.Warn("Failed to respond to the user", slog.String("error", err.Error()))
			}

			captcha.Input = ""
			editCaption = true

			defer global.Metrics.LogChatEvent("captcha_failed", captcha.ChatID, map[string]interface{}{
				"chat_id": captcha.ChatID,
				"user_id": captcha.UserID,
			})
		}

		captcha.Expiration = global.Config.Captcha.Expiration // Reset the expiration time
		if err := db.UpsertCaptcha(captcha); err != nil {
			return err
		}

		if editCaption {
			if err := c.EditCaption(
				captcha.Caption(user.Username, user.FirstName, user.LastName),
				&tele.SendOptions{
					ReplyMarkup: &tele.ReplyMarkup{
						ForceReply:     false,
						Selective:      user.Username != "",
						InlineKeyboard: captchaKeyboardDefault().keyboard,
					},
				}); err != nil {
				return err
			}

			defer global.Metrics.LogChatEvent("captcha_edited", captcha.ChatID, map[string]interface{}{
				"chat_id": captcha.ChatID,
				"user_id": captcha.UserID,
			})
		}

		return nil
	})

	return &Telegram{
		bot: bot,
	}, nil
}

// Status returns the telegram bot status.
func (t *Telegram) Status() (string, error) {
	return "ok", nil
}

// Start the bot.
func (t *Telegram) Start() {
	t.bot.Start()
}

// Get the bot user.
func (t *Telegram) Me() *model.User {
	return converters.UserFromTG(t.bot.Me).Seen()
}

// DeleteMessage deletes the message with the given chat ID and message ID.
func (t *Telegram) DeleteMessage(chatID int64, messageID int64) error {
	return t.bot.Delete(&tele.Message{
		ID:   int(messageID),
		Chat: &tele.Chat{ID: chatID},
	})
}

// Stop the bot.
func (t *Telegram) Stop() {
	t.bot.Stop()
}
