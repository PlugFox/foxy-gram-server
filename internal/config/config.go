package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the main config struct
type Config struct {
	Environment string `yaml:"environment" env:"ENVIRONMENT" env-default:"production" env-description:"Environment name"`
	Secret      string `yaml:"secret" env:"SECRET" env-default:"" env-description:"Secret key for JWT token signing and validation"`
	Verbose     string `yaml:"verbose" env:"VERBOSE" env-default:"warn" env-description:"Verbose mode for output: debug | info | warn | error"`

	Proxy    ProxyConfig    `yaml:"proxy" env-description:"Proxy SOCKS5 server config"`
	Telegram TelegramConfig `yaml:"telegram"`
	Captcha  CaptchaConfig  `yaml:"captcha"`
	API      APIConfig      `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
}

// Proxy SOCKS5 server config
type ProxyConfig struct {
	Address  string `yaml:"address" env:"PROXY_ADDRESS" env-description:"Proxy server host address"`
	Port     int    `yaml:"port" env:"PROXY_PORT" env-description:"Proxy server port"`
	Username string `yaml:"username" env:"PROXY_USERNAME" env-description:"Proxy server username"`
	Password string `yaml:"password" env:"PROXY_PASSWORD" env-description:"Proxy server password"`
}

// Telegram config
type TelegramConfig struct {
	Token     string        `yaml:"token" env:"TELEGRAM_TOKEN" env-required:"true" env-description:"Telegram bot token"`
	Timeout   time.Duration `yaml:"timeout" env:"TELEGRAM_TIMEOUT" env-default:"10s" env-description:"Telegram bot poller timeout"`
	Chats     []int64       `yaml:"chats" env:"TELEGRAM_CHATS" env-description:"Telegram chats to listen to"`
	Admins    []int64       `yaml:"admins" env:"TELEGRAM_ADMINS" env-description:"Telegram bot admins"`
	Whitelist []int64       `yaml:"whitelist" env:"TELEGRAM_WHITELIST" env-description:"Telegram bot whitelist"`
	Blacklist []int64       `yaml:"blacklist" env:"TELEGRAM_BLACKLIST" env-description:"Telegram bot blacklist"`
	IgnoreVia bool          `yaml:"ignore_via" env:"TELEGRAM_IGNORE_VIA" env-default:"false" env-description:"Ignore messages from other bots"`
}

type CaptchaConfig struct {
	Length     int           `yaml:"length" env:"CAPTCHA_LENGTH" env-default:"6" env-description:"Captcha length"`
	Width      int           `yaml:"width" env:"CAPTCHA_WIDTH" env-default:"480" env-description:"Captcha image width"`
	Height     int           `yaml:"height" env:"CAPTCHA_HEIGHT" env-default:"180" env-description:"Captcha image height"`
	Expiration time.Duration `yaml:"expiration" env:"CAPTCHA_EXPIRATION" env-default:"10m" env-description:"Captcha expiration time"`
}

// API config
type APIConfig struct {
	Host         string        `yaml:"host" env:"API_HOST" env-default:"localhost" env-description:"API host address to bind to"`
	Port         int           `yaml:"port" env:"API_PORT" env-default:"8080" env-description:"API port to bind to"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"API_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"API_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"API_IDLE_TIMEOUT" env-default:"15s"`
}

// SQLite / PostgreSQL / MySQL config for GORM dialector
type DatabaseConfig struct {
	Driver     string `yaml:"driver" env:"DATABASE_DRIVER" env-default:"sqlite3" env-description:"Database driver to use: sqlite3 | postgres | mysql"`
	Connection string `yaml:"connection" env:"DATABASE_CONNECTION" env-default:"db.sqlite3" env-description:"Connection string or path for SQLite database"`
}

// IsValid - check if the google sign in config is valid
func IsValid() bool {
	return true
}

// Error - наша собственная структура для ошибок
type Error struct {
	Message string
}

// Error - реализация метода Error для нашего типа ошибки
func (e *Error) Error() string {
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
		/* return nil, &Error{
			Message: "CONFIG_PATH is not set",
		} */
	}

	var config Config

	// Check if config file exists
	_, err := os.Stat(configPath)

	if os.IsNotExist(err) {
		err = cleanenv.ReadEnv(&config)
	} else if err == nil {
		err = cleanenv.ReadConfig(configPath, &config)
	}

	if err != nil {
		return nil, &Error{
			Message: fmt.Sprintf("Cannot read config file: %s", err),
		}
	}

	return &config, nil
}
