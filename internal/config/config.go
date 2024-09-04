package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the main config struct
type Config struct {
	Environment string         `yaml:"environment" env:"ENVIRONMENT" env-default:"production" env-description:"Environment name"`
	Secret      string         `yaml:"secret" env:"SECRET" env-default:"" env-description:"Secret key for JWT token signing and validation"`
	Verbose     string         `yaml:"verbose" env:"VERBOSE" env-default:"info" env-description:"Verbose mode for debug output"`
	Database    DatabaseConfig `yaml:"database"`
	Telegram    TelegramConfig `yaml:"telegram"`
	API         APIConfig      `yaml:"api"`
}

// Telegram config
type TelegramConfig struct {
	Token string `yaml:"token" env:"TELEGRAM_TOKEN" env-required:"true" env-description:"Telegram bot token"`
}

// API config
type APIConfig struct {
	Host         string        `yaml:"host" env:"API_HOST" env-default:"localhost" env-description:"API host address to bind to"`
	Port         int           `yaml:"port" env:"API_PORT" env-default:"8080" env-description:"API port to bind to"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"API_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"API_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"API_IDLE_TIMEOUT" env-default:"15s"`
}

// SQLite or PostgreSQL config
type DatabaseConfig struct {
	// Driver is the database driver to use. Supported drivers are "sqlite3" and "postgres".
	Driver     string `yaml:"driver" env:"DATABASE_DRIVER" env-default:"sqlite3" env-description:"Database driver to use"`
	Connection string `yaml:"connection" env:"DATABASE_CONNECTION" env-default:":memory:" env-description:"Database connection string"`
}

// IsValid - check if the google sign in config is valid
func IsValid() bool {
	return true
}

// ConfigError - наша собственная структура для ошибок
type ConfigError struct {
	Message string
}

// Error - реализация метода Error для нашего типа ошибки
func (e *ConfigError) Error() string {
	return e.Message
}

func MustLoadConfig() (*Config, error) {
	/* debugMode, err := strconv.ParseBool(os.Getenv("DEBUG_MODE"))
	if err != nil {
		debugMode = false
	} */

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yml"
		/* return nil, &ConfigError{
			Message: "CONFIG_PATH is not set",
		} */
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, &ConfigError{
			Message: fmt.Sprintf("Config file does not exist: %s", configPath),
		}
	}

	var config Config

	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		return nil, &ConfigError{
			Message: fmt.Sprintf("Cannot read config file: %s", err),
		}
	}

	return &config, nil
}
