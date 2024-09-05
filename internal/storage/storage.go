// Description: The storage package provides the database operations for the application.
// The storage package uses the GORM library to interact with the database.
//
// https://github.com/go-gorm/gorm
// https://github.com/dgraph-io/ristretto
package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dgraph-io/ristretto"
	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/errors"
	"github.com/plugfox/foxy-gram-server/internal/model"
	storage_logger "github.com/plugfox/foxy-gram-server/internal/storage/storage_logger"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Storage struct {
	cache *ristretto.Cache
	db    *gorm.DB
}

func New(config *config.Config, logger *slog.Logger) (*Storage, error) {
	// Cache
	const (
		numCounters = 1e7     // number of keys to track frequency of (10M).
		maxCost     = 1 << 28 // maximum cost of cache (256 MiB).
		bufferItems = 64      // number of keys per Get buffer.
	)
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: numCounters,
		MaxCost:     maxCost,
		BufferItems: bufferItems,
		Cost: func(value interface{}) int64 {
			switch v := value.(type) {
			case string:
				return int64(len(v)) // If string, return its length in bytes
			case []byte:
				return int64(len(v)) // If []byte, return its length
			case int, int64, float64:
				const intCost = 8
				return intCost //  int, int64, float64 - 8 bytes
			default:
				return 1 // minimal cost for other types (bool, struct, etc)
			}
		},
	})
	if err != nil {
		return nil, err
	}

	// SQL database connection
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
	if err := db.WithContext(ctx).AutoMigrate(
		&model.User{},
		&model.Chat{},
		&model.MessageOrigin{},
		&model.Message{},
		&model.ReplyMarkup{},
	); err != nil {
		return nil, err
	}

	// var result int
	// db.Raw("SELECT 1").Scan(&result)
	// logger.Debug("Result of the SELECT 1 query", slog.Int("result", result))
	// logger.Debug("Database connection established")

	return &Storage{
		cache: cache,
		db:    db,
	}, nil
}

// Close - close the database connection
func (s *Storage) Close() error {
	s.cache.Close()
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ClearCache - clear the cache
func (s *Storage) ClearCache() {
	s.cache.Clear()
}

// cacheGet - get the value from the cache
func (s *Storage) cacheGet(key string) (interface{}, bool) {
	value, ok := s.cache.Get(key)
	return value, ok
}

// cacheSet - set the value to the cache
func (s *Storage) cacheSet(key string, value interface{}) {
	s.cache.Set(key, value, 0)
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

// DeleteUser - delete the user
func (s *Storage) DeleteUser(user *model.User) error {
	return s.db.Delete(user).Error
}

// Users - get all users
func (s *Storage) Users() ([]model.User, error) {
	var users []model.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Struct for the UpsertMessage function
type UpsertMessageInput struct {
	Message *model.Message
	Chats   []*model.Chat
	Users   []*model.User
}

// UpsertMessage - insert or update the message, and the users if any
func (s *Storage) UpsertMessage(input UpsertMessageInput) error {
	if (input.Message == nil) || (input.Message.ID == 0) {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Сохраняем чаты
		if err := s.saveBatch(tx, "chat", input.Chats); err != nil {
			return err
		}

		// Сохраняем пользователей
		if err := s.saveBatch(tx, "user", input.Users); err != nil {
			return err
		}

		// Сохраняем сообщение
		if err := tx.Save(input.Message).Error; err != nil {
			return err
		}

		return nil
	})
}

// TODO: Problem with save batch

// saveBatch - сохраняем данные пачками (чаты или пользователи)
func (s *Storage) saveBatch(tx *gorm.DB, entityType string, data ...interface{}) error {
	if len(data) == 0 {
		return nil
	}

	var batchToSave []interface{}

	// Обрабатываем кэш и отбираем только те объекты, которых нет в кэше
	for _, item := range data {
		if item == nil {
			continue
		}
		var cacheKey string
		var hash string

		// Определяем тип данных для создания cacheKey
		switch entityType {
		case "chat":
			chat, ok := item.(*model.Chat)
			if !ok {
				return errors.WrapUnexpectedType("*model.Chat", item)
			}
			cacheKey = fmt.Sprintf("chat#%s", string(chat.ID))
			hash, _ = chat.Hash()
		case "user":
			user, ok := item.(*model.User)
			if !ok {
				return errors.WrapUnexpectedType("*model.User", item)
			}
			cacheKey = fmt.Sprintf("user#%s", string(user.ID))
			hash, _ = user.Hash()
		default:
			return errors.WrapUnexpectedType("unknown entityType: %s", entityType)
		}

		if hash == "" {
			return errors.WrapUnexpectedType("hash is empty", item)
		}

		// Проверяем наличие объекта в кэше
		cache, ok := s.cacheGet(cacheKey)
		if !ok || hash != cache {
			batchToSave = append(batchToSave, item)
			s.cacheSet(cacheKey, hash)
		}
	}

	// Если есть что сохранять, сохраняем всё одной пачкой
	if len(batchToSave) > 0 {
		const batchSize = 100
		if err := tx.CreateInBatches(batchToSave, batchSize).Error; err != nil {
			return err
		}
	}

	return nil
}
