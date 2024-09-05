package converters

import (
	"database/sql"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/model"
	tele "gopkg.in/telebot.v3"
)

// Convert telebot message to database message
func MessageFromTG(m *tele.Message) *model.Message {
	// Конвертация времени последнего редактирования
	var lastEdit sql.NullTime
	if m.LastEdit != 0 {
		lastEdit = sql.NullTime{
			Time:  time.Unix(m.LastEdit, 0).UTC(),
			Valid: true,
		}
	}

	// Создание объекта сообщения
	msg := &model.Message{
		ID:          model.MessageID(m.ID),
		SenderID:    model.UserID(m.Sender.ID),
		ChatID:      model.ChatID(m.Chat.ID),
		Text:        m.Text,
		Unixtime:    m.Unixtime,
		LastEdit:    lastEdit,
		Caption:     m.Caption,
		AlbumID:     m.AlbumID,
		IsForwarded: m.OriginalSender != nil,
	}

	// Если сообщение является ответом на другое сообщение
	if m.ReplyTo != nil {
		msg.ReplyToID = model.MessageID(m.ReplyTo.ID)
	}

	return msg
}

// Optional: Конвертер для пересланных сообщений
func MessageOriginFromTG(m *tele.Message) *model.MessageOrigin {
	if m.OriginalSender == nil && m.OriginalChat == nil {
		// Если сообщение не переслано
		return nil
	}

	origin := &model.MessageOrigin{
		OriginalText: m.Text,
		MessageID:    model.MessageID(m.ID),
	}

	// Если переслано от пользователя
	if m.OriginalSender != nil {
		origin.OriginalSenderID = model.UserID(m.OriginalSender.ID)
	}

	// Если переслано из чата
	if m.OriginalChat != nil {
		origin.OriginalChatID = model.ChatID(m.OriginalChat.ID)
	}

	return origin
}

func ChatFromTG(c *tele.Chat) *model.Chat {
	return &model.Chat{
		ID:        model.ChatID(c.ID),
		Type:      string(c.Type),
		Title:     c.Title,
		Username:  c.Username,
		IsPrivate: c.Private,
	}
}

// Optional: Конвертер для пользователя из сообщения
func UserFromTG(u *tele.User) *model.User {
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