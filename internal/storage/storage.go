package storage

import (
	"context"
	"log/slog"
	"time"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/model"
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
			NowFunc:        func() time.Time { return time.Now().UTC() },
		})
	if err != nil {
		return nil, err
	}

	// Migrations
	const timeoutSeconds = 15 * 60
	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses
	if err := db.WithContext(ctx).AutoMigrate(&model.User{} /* , &Channel{}, &TOTP{}, &MessageEntity{}, &MessageAttachment{}, &Message{} */); err != nil {
		return nil, err
	}

	// var result int
	// db.Raw("SELECT 1").Scan(&result)
	// logger.Debug("Result of the SELECT 1 query", slog.Int("result", result))
	// logger.Debug("Database connection established")

	return &Storage{db: db}, nil
}

// Close - close the database connection
func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// UserByID - get the user by ID
func (s *Storage) UserByID(id model.UserID) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UserByUsername - get the user by username
func (s *Storage) UserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpsertUser - insert or update the user
func (s *Storage) UpsertUser(user *model.User) error {
	return s.db.Save(user).Error
}
