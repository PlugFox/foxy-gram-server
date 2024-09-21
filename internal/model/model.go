package model

import (
	"database/sql"
	"encoding/gob"
)

// Prepare the hash function for the models to be used in the `utility.Hash(obj)` function.
func InitHashFunction() {
	// Register types for gob serialization
	gob.Register(sql.NullTime{})
	gob.Register(UserID(0))
	gob.Register(ChatID(0))
}
