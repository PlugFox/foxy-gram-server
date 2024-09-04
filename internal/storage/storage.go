package storage

import (
	"log/slog"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	storage_logger "github.com/plugfox/foxy-gram-server/internal/storage/storage_logger"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Storage struct {
	db *gorm.DB
}

func New(config *config.Config, logger *slog.Logger) (*Storage, error) {
	dialector, err := createDialector(&config.Database)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(
		dialector,
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{},
			Logger:         storage_logger.NewGormSlogLogger(logger),
		})
	if err != nil {
		return nil, err
	}

	// Migrations
	/* if err := db.AutoMigrate(&chat.User{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&chat.Channel{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&authentication.TOTP{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&MessageEntity{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&MessageAttachment{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&chat.Message{}); err != nil {
		return nil, err
	} */

	// var result int
	// db.Raw("SELECT 1").Scan(&result)
	// logger.Debug("Result of the SELECT 1 query", slog.Int("result", result))
	// logger.Debug("Database connection established")

	return &Storage{db: db}, nil
}
