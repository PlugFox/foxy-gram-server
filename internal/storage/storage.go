// Description: The storage package provides the database operations for the application.
// The storage package uses the GORM library to interact with the database.
//
// https://github.com/go-gorm/gorm
// https://github.com/dgraph-io/ristretto
package storage

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dgraph-io/ristretto"
	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/model"
	storage_logger "github.com/plugfox/foxy-gram-server/internal/storage/storagelogger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		&model.KeyValue{},
		&model.User{},
		&model.VerifiedUser{},
		&model.BannedUser{},
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

// Close - close the database connection.
func (s *Storage) Close() error {
	s.cache.Close()
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ClearCache - clear the cache.
func (s *Storage) ClearCache() {
	s.cache.Clear()
}

// cacheGet - get the value from the cache.
func (s *Storage) cacheGet(key string) (interface{}, bool) {
	value, ok := s.cache.Get(key)
	return value, ok
}

// cacheSet - set the value to the cache.
func (s *Storage) cacheSet(key string, value interface{}) {
	s.cache.Set(key, value, 0)
}

// cacheDel - delete the value from the cache.
func (s *Storage) cacheDel(key string) {
	s.cache.Del(key)
}

// UserByID - get the user by ID.
func (s *Storage) UserByID(id model.UserID) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UserByUsername - get the user by username.
func (s *Storage) UserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpsertUser - insert or update the user.
func (s *Storage) UpsertUser(user *model.User) error {
	return s.db.Save(user).Error
}

// DeleteUser - delete the user.
func (s *Storage) DeleteUser(user *model.User) error {
	return s.db.Delete(user).Error
}

// Users - get all users.
func (s *Storage) Users() ([]model.User, error) {
	var users []model.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Struct for the UpsertMessage function.
type UpsertMessageInput struct {
	Message *model.Message
	Chats   []*model.Chat
	Users   []*model.User
}

// UpsertMessage - insert or update the message, and the users if any.
func (s *Storage) UpsertMessage(input UpsertMessageInput) error {
	if (input.Message == nil) || (input.Message.ID == 0) {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Save chats
		if err := s.UpsertChats(tx, input.Chats...); err != nil {
			return err
		}

		// Save users
		if err := s.UpsertUsers(tx, input.Users...); err != nil {
			return err
		}

		// Save the message
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Save(input.Message).Error; err != nil {
			return err
		}

		return nil
	})
}

// Upsert chats if any of them have changed
//
//nolint:dupl
func (s *Storage) UpsertChats(tx *gorm.DB, data ...*model.Chat) error {
	if len(data) == 0 {
		return nil
	}

	updateCache := make(map[string]string)
	batchToSave := make(map[string]*model.Chat)

	var key string
	var hash string
	for _, chat := range data {
		if chat == nil {
			continue
		}

		key = fmt.Sprintf("_chat#%s", chat.ID.ToString())
		hash, _ = chat.Hash()
		cache, ok := s.cacheGet(key)
		if !ok || hash != cache {
			updateCache[key] = hash
			batchToSave[key] = chat
		}
	}

	// If there are chats to save
	if len(batchToSave) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(
			func() []*model.Chat {
				result := make([]*model.Chat, 0, len(batchToSave))
				for _, chat := range batchToSave {
					result = append(result, chat)
				}
				return result
			}(),
		).Error; err != nil {
			return err
		}
		for key, hash := range updateCache {
			s.cacheSet(key, hash)
		}
	}

	return nil
}

// Upsert users if any of them have changed
//
//nolint:dupl
func (s *Storage) UpsertUsers(tx *gorm.DB, data ...*model.User) error {
	if len(data) == 0 {
		return nil
	}

	updateCache := make(map[string]string)
	batchToSave := make(map[string]*model.User)

	var key string
	var hash string
	for _, user := range data {
		if user == nil {
			continue
		}

		key = fmt.Sprintf("_user#%s", user.ID.ToString())
		hash, _ = user.Hash()
		cache, ok := s.cacheGet(key)
		if !ok || hash != cache {
			updateCache[key] = hash
			batchToSave[key] = user
		}
	}

	// If there is something to save, save everything in one batch
	if len(batchToSave) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(
			func() []*model.User {
				result := make([]*model.User, 0, len(batchToSave))
				for _, user := range batchToSave {
					result = append(result, user)
				}
				return result
			}(),
		).Error; err != nil {
			return err
		}
		for key, hash := range updateCache {
			s.cacheSet(key, hash)
		}
	}

	return nil
}

// KVSet sets a key-value pair.
func (s *Storage) KVSet(key string, value interface{}) error {
	// Serialize the value using gob
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(value)
	if err != nil {
		return err
	}
	bytes := buffer.Bytes()

	// Save to cache
	s.cache.Set(fmt.Sprintf("_kv#%s", key), bytes, int64(len(bytes)))

	// Save to the database
	kv := &model.KeyValue{
		Key:   key,
		Value: bytes,
	}
	err = s.db.Clauses(clause.OnConflict{UpdateAll: true}).Save(kv).Error
	if err != nil {
		s.cache.Del(fmt.Sprintf("_kv#%s", key))
		return err
	}
	return nil
}

// KVDelete deletes a key-value pair.
func (s *Storage) KVDelete(key string) error {
	// Remove from cache
	s.cache.Del(fmt.Sprintf("_kv#%s", key))

	// Remove from the database
	return s.db.Delete(&model.KeyValue{}, "key = ?", key).Error
}

// KVGet finds a key-value pair by key.
func (s *Storage) KVGet(key string) (*model.KeyValue, error) {
	var kv model.KeyValue

	// Try to get the data from cache
	val, _ := s.cache.Get(fmt.Sprintf("_kv#%s", key))
	if val != nil {
		kv.Key = key
		kv.Value = val.([]byte)
		return &kv, nil
	}

	// If not found in cache, fetch from the database
	if err := s.db.First(&kv, "key = ?", key).Error; err != nil {
		return nil, err
	}

	// Save to cache for future access
	s.cache.Set(fmt.Sprintf("_kv#%s", key), kv.Value, int64(len(kv.Value)))

	return &kv, nil
}

// Check if the user is verified.
func (s *Storage) IsVerifiedUser(userID model.UserID) (bool, error) {
	cacheKey := fmt.Sprintf("_verified#%s", userID.ToString())
	if verified, ok := s.cacheGet(cacheKey); ok {
		return verified == true, nil
	}

	// Check existence in the database without loading the full user
	var exists bool
	err := s.db.Model(&model.VerifiedUser{}).
		Select("1").
		Where("id = ?", userID).
		Limit(1).
		Scan(&exists).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// If user is not found, cache and return false
		s.cacheSet(cacheKey, false)
		return false, nil
	} else if err != nil {
		// Return error for other issues
		return false, err
	}

	// Cache the result and return true
	s.cacheSet(cacheKey, exists)
	return exists, nil
}

// Check if the user is banned and delete expired bans.
func (s *Storage) IsBannedUser(userID model.UserID) (bool, error) {
	var bannedUser model.BannedUser
	err := s.db.Model(&model.BannedUser{}).
		Where("id = ?", userID).
		First(&bannedUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil // User is not banned
	} else if err != nil {
		return false, err // Return error for other issues
	}

	// Check if the ban has expired
	if bannedUser.ExpiresAt.Valid && bannedUser.ExpiresAt.Time.Before(time.Now()) {
		// If the ban has expired, delete the record
		if err := s.db.Delete(&bannedUser).Error; err != nil {
			return false, err
		}
		return false, nil // User is not banned anymore
	}

	// User is banned and the ban is still valid
	return true, nil
}

// Set the user as verified.
func (s *Storage) VerifyUser(verifiedUser *model.VerifiedUser) error {
	userID := verifiedUser.ID.ToString()
	s.cacheSet(fmt.Sprintf("_verified#%s", userID), true)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Remove the user from the banned list
		if err := tx.Delete(&model.BannedUser{}, "id = ?", verifiedUser.ID).Error; err != nil {
			return err
		}

		// Save the verified user
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(verifiedUser).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		s.cacheDel(fmt.Sprintf("_verified#%s", userID))
	}
	return err
}

// Ban the user.
func (s *Storage) BanUser(bannedUser *model.BannedUser) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Remove the user from the verified list
		if err := tx.Delete(&model.VerifiedUser{}, "id = ?", bannedUser.ID).Error; err != nil {
			return err
		}

		// Save the banned user
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(bannedUser).Error; err != nil {
			return err
		}
		return nil
	})
	return err
}
