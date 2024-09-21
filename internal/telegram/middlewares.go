package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
	for _, id := range config.Telegram.Chats {
		if id == chatID {
			return true
		}
	}
	return len(config.Telegram.Chats) == 0
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

// Verify the user with a local database and a CAS
func isUserBanned(db *storage.Storage, httpClient *http.Client, user *tele.User) (bool, error) {
	// Check local ban
	banned, err := db.IsBannedUser(model.UserID(user.ID))
	if err != nil {
		return false, err
	} else if banned {
		return true, nil
	}

	// Check CAS ban
	resp, err := httpClient.Get("https://api.cas.chat/check?user_id=" + user.Recipient())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Handle non-200
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the response body into an anonymous struct
	// e.g.
	// {"ok":false,"description":"Record not found."}
	// {"ok":true,"result":{"offenses":1,"messages":["..."],"time_added":"2024-09-20T18:53:39.000Z"}}
	var casResponse struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description,omitempty"`
		Result      struct {
			Offenses  int      `json:"offenses,omitempty"`
			Messages  []string `json:"messages,omitempty"`
			TimeAdded string   `json:"time_added,omitempty"`
		} `json:"result,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&casResponse)
	if err != nil {
		return false, err
	}

	// Return whether the user is flagged by CAS
	return casResponse.Ok, nil
}

// Verify user middleware - verify the user with a captcha
func verifyUserMiddleware(db *storage.Storage, httpClient *http.Client, config *config.Config, onError func(error)) tele.MiddlewareFunc {
	// Centralized error handling
	handleError := func(onError func(error), err error) {
		if onError != nil {
			onError(err)
		}
	}

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Callback() != nil {
				return next(c) // There is callback
			}

			sender := c.Sender() // Sender
			chat := c.Chat()     // Chat

			// If there is not enough parameters - skip it
			if sender.ID == 0 || chat.ID == 0 || sender.ID == chat.ID || sender.IsBot || chat.Private {
				return nil // Skip the current message
			}

			// If it not allowed chat - skip it
			if !allowedChats(config, chat.ID) {
				return nil // Skip the current message, if it is not allowed chat
			}

			// Check if it already verified user
			verified, err := db.IsVerifiedUser(model.UserID(sender.ID))
			if err != nil {
				handleError(onError, err)
				return nil // Skip the current message
			} else if verified {
				return next(c) // Verified user
			}

			// Check if the chat is valid and if the sender is an admin or the chat is private
			if chat != nil {
				member, err := c.Bot().ChatMemberOf(chat, sender)
				if err != nil {
					handleError(onError, err)
					return nil // Skip the current message
				} else if member.Role == tele.Creator || member.Role == tele.Administrator || chat.Private {
					db.VerifyUser(&model.VerifiedUser{
						ID:         model.UserID(sender.ID),
						VerifiedAt: time.Now(),
						Reason:     "Not banned",
					}) // Add user to the verification list, if it is an admin or private chat
					return next(c) // Admin or private chat - skip the verification
				}
			}

			banned, err := isUserBanned(db, httpClient, sender)
			if err != nil {
				handleError(onError, err)
				return nil // Skip the current message
			} else if banned {
				bot := c.Bot()
				// Ban the user again if they are already banned
				if err := bot.Ban(chat, &tele.ChatMember{User: sender}, true); err != nil {
					handleError(onError, err)
				}
				bot.Send(chat, fmt.Sprintf("User `%s` is banned", sender.Recipient()), tele.ModeMarkdownV2)
				return nil
			}

			// TODO: Start verification process
			// defer c.Delete() // Delete this message after processing

			// Add user to the verification list
			db.VerifyUser(&model.VerifiedUser{
				ID:         model.UserID(sender.ID),
				VerifiedAt: time.Now(),
				Reason:     "Not banned",
			})

			return next(c)
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
