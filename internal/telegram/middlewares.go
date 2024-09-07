package telegram

import (
	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/converters"
	"github.com/plugfox/foxy-gram-server/internal/model"
	"github.com/plugfox/foxy-gram-server/internal/storage"
	"github.com/plugfox/foxy-gram-server/internal/utility"
	tele "gopkg.in/telebot.v3"
)

func verifyUserMiddleware(db *storage.Storage, config *config.Config, onError func(error)) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			userId := c.Sender().ID
			chatId := c.Chat().ID
			if userId == 0 || chatId == 0 {
				return next(c) // Proceed to the next middleware if the user ID or chat ID is not available
			}

			chatFound := false
			if config.Telegram.Chats != nil && len(config.Telegram.Chats) > 0 {
				for _, id := range config.Telegram.Chats {
					if id == chatId {
						chatFound = true
					}
				}
				if !chatFound {
					return next(c) // Proceed to the next middleware if the chat is not in the list of allowed chats
				}
			}

			verified, err := db.IsVerifiedUser(model.UserID(userId))
			if err != nil && onError != nil {
				onError(err) // Log the error
			}

			if verified {
				return next(c) // Proceed to the next middleware if the user is verified
			}

			c.Delete() // Delete the message if the user is not verified

			banned, err := db.IsBannedUser(model.UserID(userId))
			if err != nil && onError != nil {
				onError(err) // Log the error
			}

			if banned {
				// Ban the user
				if err = c.Bot().Ban(c.Chat(), &tele.ChatMember{User: c.Sender()}, true); err != nil && onError != nil {
					onError(err) // Log the error
				}
				return nil // Skip the current message
			}

			// Verify flow
			captcha, err := utility.GenerateCaptcha(config.Captcha)
			if err != nil {
				if onError != nil {
					onError(err) // Log the error
				}
				return nil
			}
			photo := &tele.Photo{File: tele.FromReader(captcha)}

			// todo: reply with the captcha photo and inline keyboard

			//c.Reply(photo, tele.ModeNone, tele.NoPreview)
			/* if _, err := c.Bot().Send(c.Chat(), "You are not verified!"); err != nil && onError != nil {
				onError(err) // Log the error
			} */

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
