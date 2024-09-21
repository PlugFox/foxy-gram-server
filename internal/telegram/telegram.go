// Library repository: https://github.com/tucnak/telebot

package telegram

import (
	"log/slog"
	"net/http"

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

func New(db *storage.Storage, httpClient *http.Client, config *config.Config, logger *slog.Logger) (*Telegram, error) {
	pref := tele.Settings{
		Token:  config.Telegram.Token,
		Client: httpClient,
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

	bot.Use(verifyUserMiddleware(db, httpClient, config, func(err error) {
		logger.Error("verify user error", slog.String("error", err.Error()))
	}))
	if config.Telegram.Whitelist != nil && len(config.Telegram.Whitelist) > 0 {
		bot.Use(mw.Whitelist(config.Telegram.Whitelist...))
	}
	if config.Telegram.Blacklist != nil && len(config.Telegram.Blacklist) > 0 {
		bot.Use(mw.Blacklist(config.Telegram.Blacklist...))
	}

	// Store messages in the database
	bot.Use(storeMessagesMiddleware(db, func(err error) {
		logger.Error("store message error", slog.String("error", err.Error()))
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

	/* bot.Handle("/id", func(c tele.Context) error {
		c.Reply(fmt.Sprintf("Your ID: `%d`\nChat ID: `%d`", c.Sender().ID, c.Chat().ID), tele.ModeMarkdownV2)
		return nil
	}) */

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

// Start the bot
func (t *Telegram) Start() {
	t.bot.Start()
}

// Get the bot user
func (t *Telegram) Me() *model.User {
	return converters.UserFromTG(t.bot.Me).Seen()
}

// Stop the bot
func (t *Telegram) Stop() {
	t.bot.Stop()
}
