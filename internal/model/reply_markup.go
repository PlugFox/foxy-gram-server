package model

type ReplyMarkup struct {
	ID        int64     `gorm:"PrimaryKey" json:"id"`
	MessageID MessageID `gorm:"index"      json:"message_id"`
	Data      string    `json:"data"` // JSON-encoded reply markup.

	// Relations
	Message *Message `gorm:"foreignKey:MessageID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// TableName - set the table name.
func (ReplyMarkup) TableName() string {
	return "reply_markups"
}
