// Library repository: https://github.com/tucnak/telebot

package telegram

import (
	"log/slog"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/converters"
	"github.com/plugfox/foxy-gram-server/internal/model"
	"github.com/plugfox/foxy-gram-server/internal/storage"

	log "github.com/plugfox/foxy-gram-server/internal/log"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	mw "gopkg.in/telebot.v3/middleware"
)

type Telegram struct {
	bot *tele.Bot
}

func New(db *storage.Storage, config *config.Config, logger *slog.Logger) (*Telegram, error) {
	// todo: restore last id from the database
	pref := tele.Settings{
		Token: config.Telegram.Token,
		Poller: &tele.LongPoller{
			Timeout: config.Telegram.Timeout,
		},
		OnError: func(err error, _ tele.Context) {
			logger.Error("telegram error", slog.String("error", err.Error()))
		},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	// Global-scoped middleware:
	bot.Use(mw.Recover())
	bot.Use(mw.AutoRespond())
	bot.Use(mw.Logger(log.NewLogAdapter(logger)))
	if config.Telegram.IgnoreVia {
		bot.Use(mw.IgnoreVia())
	}
	if config.Telegram.Whitelist != nil && len(config.Telegram.Whitelist) > 0 {
		bot.Use(mw.Whitelist(config.Telegram.Whitelist...))
	}
	if config.Telegram.Blacklist != nil && len(config.Telegram.Blacklist) > 0 {
		bot.Use(mw.Blacklist(config.Telegram.Blacklist...))
	}

	// Store messages in the database
	bot.Use(storeMessagesMiddleware(db, func(err error) {
		logger.Error("database error", slog.String("error", err.Error()))
	}))

	/* bot.Use(mw.Restrict(mw.RestrictConfig{
		Chats: []int64{config.Telegram.ChatID},
		In: func(c tele.Context) error {
			return c.Send("Hello!")
		},
		Out: func(c tele.Context) error {
			return c.Send("Sorry, I don't know you")
		},
	})) */

	/* bot.Handle("/hello", func(c tele.Context) error {
		return c.Send("Hello!")
	}) */

	// Group-scoped middleware:
	if config.Telegram.Admins != nil && len(config.Telegram.Admins) > 0 {
		adminOnly := bot.Group()
		adminOnly.Use(middleware.Whitelist(config.Telegram.Admins...))
		/* adminOnly.Handle("/ban", onBan)
		adminOnly.Handle("/kick", onKick) */
	}

	// TODO: add more handlers
	// tele.OnAddedToGroup
	// tele.OnUserJoined
	// Verify the user is passing the captcha or sending the code with buttons
	// check out examples at the github

	// On text message
	bot.Handle(tele.OnText, func(_ tele.Context) error {
		// c.Reply("Hello!")
		return nil
	})

	// On edited message
	bot.Handle(tele.OnEdited, func(_ tele.Context) error {
		// c.Reply("Hello!")
		return nil
	})

	return &Telegram{
		bot: bot,
	}, nil
}

func (t *Telegram) Start() {
	t.bot.Start()
}

func (t *Telegram) Me() *model.User {
	return converters.UserFromTG(t.bot.Me).Seen()
}

func (t *Telegram) Stop() {
	t.bot.Stop()
}

// storeMessages middleware - store messages in the database asynchronously
func storeMessagesMiddleware(db *storage.Storage, onError func(error)) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			msg := c.Message()
			if msg != nil {
				go func() {
					// TODO: save chat info too, not only the message and user
					// pass a structure to the function

					// TODO: update last message id in the database

					// TODO: create a in memmory cache for the last message id and other data
					err := db.UpsertMessage(
						storage.UpsertMessageInput{
							Message: converters.MessageFromTG(msg),
							Chats: []*model.Chat{
								converters.ChatFromTG(msg.Chat),
								// converters.ChatFromTG(msg.SenderChat),
								// converters.ChatFromTG(msg.OriginalChat),
							}, Users: []*model.User{
								converters.UserFromTG(msg.Sender).Seen(),
								// converters.UserFromTG(msg.OriginalSender),
								// converters.UserFromTG(msg.Via),
								// converters.UserFromTG(msg.UserJoined),
								// converters.UserFromTG(msg.UserLeft),
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

func convertUser(u *tele.User) *model.User {
	return &model.User{
		ID:           model.UserID(u.ID),
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		Username:     u.Username,
		LanguageCode: u.LanguageCode,
		IsPremium:    u.IsPremium,
		IsBot:        u.IsBot,
	}
}
