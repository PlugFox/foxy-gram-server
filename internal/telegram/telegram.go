// Library repository: https://github.com/tucnak/telebot

package telegram

import (
	"log/slog"
	"net/http"

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

func New(db *storage.Storage, httpClient *http.Client) (*Telegram, error) {
	pref := tele.Settings{
		Token:  global.Config.Telegram.Token,
		Client: httpClient,
		Poller: &tele.LongPoller{
			Timeout: global.Config.Telegram.Timeout,
		},
		OnError: func(err error, _ tele.Context) {
			global.Logger.Error("telegram error", slog.String("error", err.Error()))
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

	bot.Use(verifyUserWithCAS(db, httpClient, func(err error) {
		global.Logger.Error("verify user with cas error", slog.String("error", err.Error()))
	}))

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
	if len(global.Config.Telegram.Admins) > 0 {
		adminOnly := bot.Group()
		/* adminOnly.Handle("/ban", onBan)
		adminOnly.Handle("/kick", onKick) */
		adminOnly.Use(middleware.Whitelist(global.Config.Telegram.Admins...))
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

	// TODO: handle captcha methods, get information about captcha directly from the database

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
