package model

import (
	"database/sql"
	"time"

	tele "gopkg.in/telebot.v3"
	"gorm.io/gorm"
)

type (
	MessageID int64
)

type Message struct {
	ID          MessageID    `gorm:"PrimaryKey" json:"id"`     // Unique message identifier.
	SenderID    UserID       `gorm:"index" json:"sender_id"`   // ID of the sender.
	ChatID      ChatID       `gorm:"index" json:"chat_id"`     // ID of the chat the message belongs to.
	Text        string       `json:"text"`                     // Message text.
	Unixtime    int64        `json:"unixtime"`                 // Unix timestamp when the message was sent.
	LastEdit    sql.NullTime `json:"last_edit"`                // Time of last edit.
	AlbumID     string       `json:"album_id"`                 // Optional. ID of the media album the message belongs to.
	Caption     string       `json:"caption"`                  // Optional. Media caption.
	IsForwarded bool         `json:"is_forwarded"`             // True if the message was forwarded.
	ReplyToID   MessageID    `gorm:"index" json:"reply_to_id"` // Optional. ID of the original message for replies.

	// Meta fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"` // Time when the message was stored.
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"` // Time when the message was last updated.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`          // Soft delete.

	// Relations
	Sender  *User    `gorm:"foreignKey:SenderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`  // Reference to the sender.
	Chat    *Chat    `gorm:"foreignKey:ChatID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`    // Reference to the chat.
	ReplyTo *Message `gorm:"foreignKey:ReplyToID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // Reply message (self-reference).
}

// Create a new message from the telegram message
func MessageFromTG(m *tele.Message) *Message {
	return &Message{
		ID:          MessageID(m.ID),
		SenderID:    UserID(m.Sender.ID),
		ChatID:      ChatID(m.Chat.ID),
		Text:        m.Text,
		Unixtime:    m.Unixtime,
		Caption:     m.Caption,
		AlbumID:     m.AlbumID,
		IsForwarded: m.OriginalSender != nil,
	}
}

// TableName - set the table name
func (Message) TableName() string {
	return "messages"
}
