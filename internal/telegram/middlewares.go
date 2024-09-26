package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/converters"
	"github.com/plugfox/foxy-gram-server/internal/global"
	"github.com/plugfox/foxy-gram-server/internal/model"
	"github.com/plugfox/foxy-gram-server/internal/storage"
	tele "gopkg.in/telebot.v3"
)

const (
	contextKeyShouldVerify = "should_verify" // Context key for the verification flag, we should verify the user
)

var errorUnexpectedStatusCode = fmt.Errorf("unexpected status code")

// Check if the chat is allowed.
func allowedChats(chatID int64) bool {
	for _, id := range global.Config.Telegram.Chats {
		if id == chatID {
			return true
		}
	}

	return len(global.Config.Telegram.Chats) == 0
}

// Restrict user rights
//
//nolint:unused
func restrictUser(bot *tele.Bot, chat *tele.Chat, user *tele.User, rights tele.Rights, until time.Time) error {
	return bot.Restrict(chat, &tele.ChatMember{
		User:            user,
		Rights:          rights,
		RestrictedUntil: until.Unix(),
	})
}

// Kick user from the chat (ban) for 1 hour
//
//nolint:unused
func kickUser(bot *tele.Bot, chat *tele.Chat, user *tele.User) error {
	return bot.Ban(chat, &tele.ChatMember{
		User:            user,
		RestrictedUntil: time.Now().Add(time.Hour).Unix(),
	}, true)
}

// Verify the user with a local database
func isUserLocalBanned(db *storage.Storage, user *tele.User) (bool, error) {
	// Check local ban
	banned, err := db.IsBannedUser(model.UserID(user.ID))
	if err != nil {
		return false, err
	} else if banned {
		return true, nil
	}

	return false, nil
}

// Verify the user with a CAS ban
func isUserCASBanned(httpClient *http.Client, user *tele.User) (bool, error) {
	// Check CAS ban
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet, "https://api.cas.chat/check?user_id="+user.Recipient(),
		nil,
	)
	if err != nil {
		return false, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, errorUnexpectedStatusCode
	}
	defer resp.Body.Close()

	// Handle non-200
	if resp.StatusCode != http.StatusOK {
		return false, errorUnexpectedStatusCode
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
//
//nolint:cyclop,gocognit
func verifyUserMiddleware(
	db *storage.Storage,
	onError func(error),
) tele.MiddlewareFunc {
	// Centralized error handling
	handleError := func(err error) {
		if onError != nil {
			onError(err)
		}
	}

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			c.Set(contextKeyShouldVerify, true)

			if c.Callback() != nil {
				c.Set(contextKeyShouldVerify, false) // Skip the verification for callbacks

				return next(c) // There is callback
			}

			sender := c.Sender() // Sender
			chat := c.Chat()     // Chat

			// If there is not enough parameters - skip it
			if sender == nil || chat == nil || sender.ID == 0 || chat.ID == 0 || sender.ID == chat.ID || sender.IsBot || chat.Private {
				return nil // Skip the current message
			}

			// If it not allowed chat - skip it
			if !allowedChats(chat.ID) {
				return nil // Skip the current message, if it is not allowed chat
			}

			// Check if it already verified user
			verified, err := db.IsVerifiedUser(model.UserID(sender.ID))
			if err != nil {
				handleError(err)

				return nil // Skip the current message
			} else if verified {
				c.Set(contextKeyShouldVerify, false) // Skip the verification for callbacks

				return next(c) // Verified user
			}

			// Check if the chat is valid and if the sender is an admin or the chat is private
			member, err := c.Bot().ChatMemberOf(chat, sender)
			if err != nil {
				handleError(err)

				return nil // Skip the current message
			} else if member.Role == tele.Creator || member.Role == tele.Administrator || chat.Private {
				// Add user to the verification list, if it is an admin or private ch
				if err := db.VerifyUser(&model.VerifiedUser{
					ID:         model.UserID(sender.ID),
					VerifiedAt: time.Now(),
					Reason:     "Not banned",
				}); err != nil {
					handleError(err)
				}

				c.Set(contextKeyShouldVerify, false) // Skip the verification, because the user is an admin

				return next(c) // Admin or private chat - skip the verification
			}

			c.Set(contextKeyShouldVerify, true) // Should verify the user

			// Delete the current message, because user is not verified
			if err := c.Delete(); err != nil {
				handleError(err)
			}

			return next(c)
		}
	}
}

// Verify the user with a local database
func verifyUserWithLocalDB(
	db *storage.Storage,
	onError func(error),
) tele.MiddlewareFunc {
	// Centralized error handling
	handleError := func(err error) {
		if onError != nil {
			onError(err)
		}
	}

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Get(contextKeyShouldVerify) != true {
				return next(c) // Skip the verification for callbacks
			}

			banned, err := isUserLocalBanned(db, c.Sender())
			if err != nil {
				handleError(err)

				return nil // Skip the current message
			} else if banned {
				bot := c.Bot()
				// Ban the user again if they are already banned
				if err := bot.Ban(c.Chat(), &tele.ChatMember{User: c.Sender()}, true); err != nil {
					handleError(err)
				}

				// Send the message to the chat
				msg := fmt.Sprintf("User `%s` is banned in local db", c.Sender().Recipient())
				if _, err := bot.Send(c.Chat(), msg, tele.ModeMarkdownV2); err != nil {
					handleError(err)
				}

				return nil // Skip the next pipeline
			}

			return next(c) // Continue the pipeline
		}
	}
}

// Verify the user with a CAS ban
func verifyUserWithCAS(
	db *storage.Storage,
	httpClient *http.Client,
	onError func(error),
) tele.MiddlewareFunc {
	// Centralized error handling
	handleError := func(err error) {
		if onError != nil {
			onError(err)
		}
	}

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Get(contextKeyShouldVerify) != true {
				return next(c) // Skip the verification for callbacks
			}

			banned, err := isUserCASBanned(httpClient, c.Sender())
			if err != nil {
				handleError(err)

				return nil // Skip the current message
			} else if banned {
				bot := c.Bot()
				// Ban the user again if they are already banned
				if err := bot.Ban(c.Chat(), &tele.ChatMember{User: c.Sender()}, true); err != nil {
					handleError(err)
				}

				// Send the message to the chat
				msg := fmt.Sprintf("User `%s` is CAS banned", c.Sender().Recipient())
				if _, err := bot.Send(c.Chat(), msg, tele.ModeMarkdownV2); err != nil {
					handleError(err)
				}

				// Ban the user in the local database
				if err := db.BanUser(&model.BannedUser{
					ID:       model.UserID(c.Sender().ID),
					BannedAt: time.Now(),
					Reason:   "CAS banned",
				}); err != nil {
					handleError(err)
				}

				return nil // Skip the next pipeline
			}

			return next(c) // Continue the pipeline
		}
	}
}

// Verify the user with a captcha
func verifyUserWithCaptcha(
	db *storage.Storage,
	onError func(error),
) tele.MiddlewareFunc {
	// Centralized error handling
	handleError := func(err error) {
		if onError != nil {
			onError(err)
		}
	}

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Get(contextKeyShouldVerify) != true {
				return next(c) // Skip the verification for callbacks
			}

			captchas, err := db.GetCaptchasForUserID(c.Sender().ID, c.Chat().ID)
			if err != nil {
				handleError(err)
			}
			if len(captchas) > 0 {
				return nil // User already has a captcha
			}

			// Create a new captcha
			buffer := new(bytes.Buffer)
			captcha, err := model.GenerateCaptcha(buffer)
			if err != nil {
				handleError(err)

				return nil // Skip the current message
			}

			// Send the captcha message
			bot := c.Bot()
			reply, err := bot.Send(c.Chat(), tele.Photo{
				File:    tele.FromReader(buffer),
				Width:   captcha.Width,
				Height:  captcha.Height,
				Caption: fmt.Sprintf("Please solve the captcha."),
			}, tele.ReplyMarkup{
				ForceReply: true,
				/* Selective:  c.Sender().Username != "", */
				InlineKeyboard: [][]tele.InlineButton{
					{},
					{},
					{},
					{},
					{
						tele.InlineButton{Text: "Refresh 🔄", Unique: "refresh_captcha"},
						tele.InlineButton{Text: "Cancel ❌", Unique: "cancel_captcha"},
					},
				},
			})
			buffer.Reset()

			captcha.UserID = c.Sender().ID
			captcha.ChatID = reply.Chat.ID
			captcha.MessageID = int64(reply.ID)

			db.UpsertCaptcha(captcha) // Upsert the captcha to the database

			return nil

			/* type captchaMessage struct {
				buffer  *bytes.Buffer
				captcha *model.Captcha
				photo   tele.Photo
				reply   tele.ReplyMarkup
			} */

			// return next(c) // Continue the pipeline
		}
	}
}

// storeMessages middleware - store messages in the database asynchronously.
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
/* func verifyUserWithCaptcha(
	channel chan error,
	db *storage.Storage,
	config *config.Config,
	bot *tele.Bot,
	chat *tele.Chat,
	user *tele.User,
) {
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

// Add user to the verification list
/* if err := db.VerifyUser(&model.VerifiedUser{
	ID:         model.UserID(sender.ID),
	VerifiedAt: time.Now(),
	Reason:     "Not banned",
}); err != nil {
	handleError(onError, err)
} */
