package telegram

import (
	"bytes"
	"time"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/converters"
	"github.com/plugfox/foxy-gram-server/internal/model"
	"github.com/plugfox/foxy-gram-server/internal/storage"
	"github.com/plugfox/foxy-gram-server/internal/utility"
	tele "gopkg.in/telebot.v3"
)

type captchaMessage struct {
	buffer  *bytes.Buffer
	captcha *utility.Captcha
	photo   tele.Photo
	reply   tele.ReplyMarkup
}

// Check if the chat is allowed
func allowedChats(config *config.Config, chatID int64) bool {
	if config.Telegram.Chats == nil || len(config.Telegram.Chats) == 0 {
		return true
	}
	for _, id := range config.Telegram.Chats {
		if id == chatID {
			return true
		}
	}
	return false
}

// Restrict user rights
func restrictUser(bot *tele.Bot, chat *tele.Chat, user *tele.User, rights tele.Rights, until time.Time) error {
	return bot.Restrict(chat, &tele.ChatMember{
		User:            user,
		Rights:          rights,
		RestrictedUntil: until.Unix(),
	})
}

// Kick user from the chat (ban) for 1 hour
func kickUser(bot *tele.Bot, chat *tele.Chat, user *tele.User) error {
	return bot.Ban(chat, &tele.ChatMember{
		User:            user,
		RestrictedUntil: time.Now().Add(time.Hour).Unix(),
	}, true)
}

// Verify the user with a captcha
/* func verifyUserWithCaptcha(channel chan error, db *storage.Storage, config *config.Config, bot *tele.Bot, chat *tele.Chat, user *tele.User) {
	banned, err := db.IsBannedUser(model.UserID(user.ID))
	if err != nil {
		channel <- err
		return
	}

	if banned {
		// Ban the user again if they are already banned
		if err = bot.Ban(chat, &tele.ChatMember{User: user}, true); err != nil {
			channel <- err
		}
		close(channel)
		return // Skip the current message
	}

	// Restrict the user from sending messages
	if err := restrictUser(bot, chat, user, tele.Rights{
		CanSendMessages: false,
		CanSendMedia:    false,
		CanSendOther:    false,
	}, time.Now().Add(config.Captcha.Expiration)); err != nil {
		channel <- err
	}

	// Build the captcha message with the reply markup
	captchaMessage, err := buildCaptchaMessage(config.Captcha, *user)
	if err != nil {
		channel <- err
		close(channel)
		return
	}

	// Send the captcha message
	reply, err := bot.Send(chat, captchaMessage.photo, captchaMessage.reply)
	captchaMessage.buffer.Reset()
	if err != nil {
		channel <- err
		close(channel)
		return
	}

	// Schedule the deletion of the captcha message
	timer := time.AfterFunc(captchaMessage.captcha.Expiration, func() {
		if err := bot.Delete(reply); err != nil {
			channel <- err
		}
	})

	// Handle button events
	bot.Handle(&cancelBtn, func(c tele.Context) error {
		if user.ID != c.Sender().ID {
			if err := c.Respond(&tele.CallbackResponse{
				Text:      "Only the sender can cancel the captcha.",
				ShowAlert: false,
			}); err != nil {
				channel <- err
			}
			return nil // Skip the current event if the sender is not the same
		}
		timer.Stop() // Stop the deletion timer
		if err := bot.Delete(reply); err != nil {
			channel <- err
		}
		if err := c.Respond(&tele.CallbackResponse{
			Text:      "Captcha canceled.",
			ShowAlert: false,
		}); err != nil {
			channel <- err
		}
		return nil
	})

	// Handle the refresh button
	bot.Handle(&refreshBtn, func(c tele.Context) error {
		if user.ID != c.Sender().ID {
			if err := c.Respond(&tele.CallbackResponse{
				Text:      "Only the sender can refresh the captcha.",
				ShowAlert: false,
			}); err != nil {
				channel <- err
			}
			return nil // Skip the current event if the sender is not the same
		}
		timer.Stop() // Stop the deletion timer
		captchaBuffer := new(bytes.Buffer)
		defer captchaBuffer.Reset()
		if err := captchaPtr.Refresh(captchaBuffer); err != nil {
			channel <- err
			return nil
		}
		if err := c.Edit(&tele.Photo{
			File:   tele.FromReader(captchaBuffer),
			Width:  captchaPtr.Width,
			Height: captchaPtr.Height,
		}, &tele.ReplyMarkup{
			ForceReply: true,
			Selective:  user.Username != "",
			InlineKeyboard: [][]tele.InlineButton{
				{cancelBtn, refreshBtn},
				{
					tele.InlineButton{Text: "12", Unique: "1"},
					tele.InlineButton{Text: "34", Unique: "2"},
				},
				{
					tele.InlineButton{Text: "56", Unique: "3"},
					tele.InlineButton{Text: "78", Unique: "4"},
				},
			},
		}); err != nil {
			channel <- err
			return nil
		}
		timer.Reset(captchaPtr.Expiration) // Reset the deletion timer
		if err := c.Respond(&tele.CallbackResponse{
			Text:      "Captcha refreshed.",
			ShowAlert: false,
		}); err != nil {
			channel <- err
		}
		return nil
	})
} */

// Verify the user with a CAS
func verifyUserWithCAS(channel chan error, db *storage.Storage, config *config.Config, bot *tele.Bot, chat *tele.Chat, user *tele.User) {
}

// Verify user middleware - verify the user with a captcha
func verifyUserMiddleware(db *storage.Storage, config *config.Config, onError func(error)) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Callback() != nil {
				return next(c) // Thats a callback, proceed to the next middleware / handler
			}

			sender := c.Sender()
			chat := c.Chat()

			// Check if the user is admin or the chat is private
			/* if chat != nil {
				member, err := c.Bot().ChatMemberOf(chat, sender)
				if err == nil {
					isAdmin := member.Role == tele.Creator || member.Role == tele.Administrator
					if isAdmin || chat.Private {
						return next(c) // Proceed to the next middleware if the user is admin or the chat is private
					}
				}
			} */

			if sender.ID == 0 || chat.ID == 0 || sender.ID == chat.ID || sender.IsBot || chat.Private {
				return nil // Ignore if the user ID or chat ID is not available or thats a PM
			}

			verified, err := db.IsVerifiedUser(model.UserID(sender.ID))
			if err != nil /* && onError != nil */ {
				if onError != nil {
					onError(err) // Log the error
				}
				return nil // Skip the current message
			} else if verified {
				return next(c) // Proceed to the next middleware if the user is verified
			}

			// Should we verify the user in this chat?
			if !allowedChats(config, chat.ID) {
				return nil // Ignore if the chat is not in the allowed chats list
			}

			// Verify the user asynchronously
			defer c.Delete() // Delete the message, because the user is not verified

			bot := c.Bot()

			banned, err := db.IsBannedUser(model.UserID(sender.ID))
			if err != nil {
				if onError != nil {
					onError(err) // Log the error
				}
				return nil
			} else if banned {
				// Ban the user again if they are already banned
				err = bot.Ban(chat, &tele.ChatMember{User: sender}, true)
				if err != nil {
					if onError != nil {
						onError(err) // Log the error
					}
					return nil
				}
				return nil // Skip the current message
			}

			channel := make(chan error)
			go verifyUserWithCAS(channel, db, config, bot, chat, sender)
			select {
			case err := <-channel:
				if err != nil && onError != nil {
					onError(err) // Log the error
				}
			}
			return nil // Skip the current message
		}
	}
}

// storeMessages middleware - store messages in the database asynchronously
func storeMessagesMiddleware(db *storage.Storage, onError func(error)) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			msg := c.Message()
			if msg != nil {
				go func() {
					err := db.UpsertMessage(
						storage.UpsertMessageInput{
							Message: converters.MessageFromTG(msg),
							Chats: []*model.Chat{
								converters.ChatFromTG(msg.Chat),
								converters.ChatFromTG(msg.SenderChat),
								converters.ChatFromTG(msg.OriginalChat),
							}, Users: []*model.User{
								converters.UserFromTG(msg.Sender).Seen(),
								converters.UserFromTG(msg.OriginalSender),
								converters.UserFromTG(msg.Via),
								converters.UserFromTG(msg.UserJoined),
								converters.UserFromTG(msg.UserLeft),
							},
						})
					if err != nil && onError != nil {
						onError(err)
					}
				}()
			}
			return next(c)
		}
	}
}
