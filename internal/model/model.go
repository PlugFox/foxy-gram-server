package model

import (
	"database/sql"
	"encoding/gob"
)

func InitHashFunction() {
	// Register types for gob serialization
	gob.Register(sql.NullTime{})
	gob.Register(UserID(0))
	gob.Register(ChatID(0))
}
