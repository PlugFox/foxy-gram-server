// https://github.com/tucnak/telebot

package telegram

import (
	"log/slog"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/model"

	log "github.com/plugfox/foxy-gram-server/internal/log"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	mw "gopkg.in/telebot.v3/middleware"
)

type Telegram struct {
	bot *tele.Bot
}

func New(config *config.Config, logger *slog.Logger) (*Telegram, error) {
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

	return &Telegram{
		bot: bot,
	}, nil
}

func (t *Telegram) Start() {
	t.bot.Start()
}

func (t *Telegram) Me() *model.User {
	return convertUser(t.bot.Me).Seen()
}

func (t *Telegram) Stop() {
	t.bot.Stop()
}

func convertUser(u *tele.User) *model.User {
	return &model.User{
		ID:           model.UserID(u.ID),
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		Username:     u.Username,
		Usernames:    u.Usernames,
		LanguageCode: u.LanguageCode,
		IsPremium:    u.IsPremium,
		IsBot:        u.IsBot,
	}
}
