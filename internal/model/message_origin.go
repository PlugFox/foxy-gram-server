package model

type MessageOrigin struct {
	ID               int64     `gorm:"PrimaryKey"    json:"id"`            // Unique identifier for the message origin.
	OriginalChatID   ChatID    `gorm:"index"         json:"original_chat_id"`   // ID of the original chat.
	MessageID        MessageID `gorm:"index"         json:"message_id"`         // ID of the forwarded message.
	OriginalSenderID UserID    `gorm:"index"         json:"original_sender_id"` // ID of the original sender.
	OriginalText     string    `json:"original_text"`                   // Text of the original message.

	// Relations
	OriginalChat   *Chat    `gorm:"foreignKey:OriginalChatID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`   // Reference to the original chat.
	OriginalSender *User    `gorm:"foreignKey:OriginalSenderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // Reference to the original sender.
	Message        *Message `gorm:"foreignKey:MessageID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`        // Reference to the original message.
}

// TableName - set the table name.
func (MessageOrigin) TableName() string {
	return "message_origins"
}
