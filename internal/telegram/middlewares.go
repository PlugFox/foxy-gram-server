package telegram

import (
	"bytes"
	"fmt"
	"time"

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
			if c.Callback() != nil {
				return next(c) // Thats a callback, proceed to the next middleware / handler
			}

			sender := c.Sender()
			chat := c.Chat()

			if sender.ID == 0 || chat.ID == 0 || sender.ID == chat.ID || sender.IsBot || chat.Private {
				return nil // Ignore if the user ID or chat ID is not available or thats a PM
			}

			verified, err := db.IsVerifiedUser(model.UserID(sender.ID))
			if err != nil && onError != nil {
				onError(err) // Log the error
			} else if verified {
				return next(c) // Proceed to the next middleware if the user is verified
			}

			// Should we verify the user in this chat?
			if config.Telegram.Chats != nil && len(config.Telegram.Chats) > 0 {
				found := false
				for _, id := range config.Telegram.Chats {
					if id != chat.ID {
						found = true
						break
					}
				}
				if !found {
					return nil // Skip the current message if the chat is not in the list
				}
			}

			// Verify the user asynchronously
			defer c.Delete() // Delete the message, because the user is not verified
			bot := c.Bot()
			// todo: refactoring, add buttons with captcha emojies callback data
			// Kick user if make wrong answer
			go func() {
				banned, err := db.IsBannedUser(model.UserID(sender.ID))
				if err != nil && onError != nil {
					onError(err) // Log the error
				}

				if banned {
					// Ban the user
					if err = bot.Ban(chat, &tele.ChatMember{User: sender}, true); err != nil && onError != nil {
						onError(err) // Log the error
					}
					return // Skip the current message
				}

				// Verify flow: reply with the captcha photo and inline keyboard
				var caption string
				if username := sender.Username; username != "" {
					caption = fmt.Sprintf("@%s, please solve the captcha.\nReply with the code in the image.", username)
				} else if firstName := sender.FirstName; firstName != "" {
					caption = "%s, please solve the captcha.\nReply with the code in the image."
				} else {
					caption = "Please solve the captcha.\nReply with the code in the image."
				}
				captchaBuffer := new(bytes.Buffer)
				defer captchaBuffer.Reset()
				captchaPtr, err := utility.GenerateCaptcha(config.Captcha, captchaBuffer)
				if err != nil {
					if onError != nil {
						onError(err) // Log the error
					}
					return // Skip the current message
				}

				refreshBtn := tele.InlineButton{Text: "Refresh 🔄", Unique: "refresh_captcha"}
				cancelBtn := tele.InlineButton{Text: "Cancel ❌", Unique: "cancel_captcha"}

				reply, err := bot.Send(chat, &tele.Photo{
					File:    tele.FromReader(captchaBuffer),
					Width:   captchaPtr.Width,
					Height:  captchaPtr.Height,
					Caption: caption,
				}, &tele.SendOptions{
					ReplyMarkup: &tele.ReplyMarkup{
						ForceReply: true,
						Selective:  sender.Username != "",
						InlineKeyboard: [][]tele.InlineButton{
							{cancelBtn, refreshBtn},
							{
								tele.InlineButton{Text: "12", Unique: "1", Data: "12"},
								tele.InlineButton{Text: "34", Unique: "2", Data: "34"},
							},
							{
								tele.InlineButton{Text: "56", Unique: "3", Data: "56"},
								tele.InlineButton{Text: "78", Unique: "4", Data: "78"},
							},
						},
					},
				})
				if err != nil && onError != nil {
					onError(err) // Log the error
				}

				// Schedule the deletion of the captcha message
				timer := time.AfterFunc(captchaPtr.Expiration, func() {
					bot.Delete(reply)
				})

				// Handle button events
				bot.Handle(&cancelBtn, func(c tele.Context) error {
					if sender.ID != c.Sender().ID {
						c.Respond(&tele.CallbackResponse{
							Text:      "Only the sender can cancel the captcha.",
							ShowAlert: false,
						})
						return nil // Skip the current event if the sender is not the same
					}
					timer.Stop() // Stop the deletion timer
					c.Delete()   // Delete the captcha message
					c.Respond(&tele.CallbackResponse{
						Text:      "Captcha canceled.",
						ShowAlert: false,
					})
					return nil
				})

				// Handle the refresh button
				bot.Handle(&refreshBtn, func(c tele.Context) error {
					if sender.ID != c.Sender().ID {
						c.Respond(&tele.CallbackResponse{
							Text:      "Only the sender can refresh the captcha.",
							ShowAlert: false,
						})
						return nil // Skip the current event if the sender is not the same
					}
					timer.Stop() // Stop the deletion timer
					captchaBuffer := new(bytes.Buffer)
					defer captchaBuffer.Reset()
					if err := captchaPtr.Refresh(captchaBuffer); err != nil {
						if onError != nil {
							onError(err) // Log the error
						}
						return nil
					}
					c.Edit(&tele.Photo{
						File:   tele.FromReader(captchaBuffer),
						Width:  captchaPtr.Width,
						Height: captchaPtr.Height,
						/* Caption: caption, */
					}, &tele.ReplyMarkup{
						ForceReply: true,
						Selective:  sender.Username != "",
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
					})
					timer.Reset(captchaPtr.Expiration) // Reset the deletion timer
					c.Respond(&tele.CallbackResponse{
						Text:      "Captcha refreshed.",
						ShowAlert: false,
					})
					return nil
				})
			}()

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
