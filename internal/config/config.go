package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the main config struct.
type Config struct {
	Environment string `env:"ENVIRONMENT" env-default:"production" env-description:"Environment name"                                     yaml:"environment"`
	Secret      string `env:"SECRET"      env-default:""           env-description:"Secret key for JWT token signing and validation"      yaml:"secret"`
	Verbose     string `env:"VERBOSE"     env-default:"warn"       env-description:"Verbose mode for output: debug | info | warn | error" yaml:"verbose"`

	Proxy    ProxyConfig    `env-description:"Proxy SOCKS5 server config" yaml:"proxy"`
	Telegram TelegramConfig `yaml:"telegram"`
	Captcha  CaptchaConfig  `yaml:"captcha"`
	API      APIConfig      `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
}

// Proxy SOCKS5 server config.
type ProxyConfig struct {
	Address  string `env:"PROXY_ADDRESS"  env-description:"Proxy server host address" yaml:"address"`
	Port     int    `env:"PROXY_PORT"     env-description:"Proxy server port"         yaml:"port"`
	Username string `env:"PROXY_USERNAME" env-description:"Proxy server username"     yaml:"username"`
	Password string `env:"PROXY_PASSWORD" env-description:"Proxy server password"     yaml:"password"`
}

// Telegram config.
type TelegramConfig struct {
	Token     string        `env:"TELEGRAM_TOKEN"      env-description:"Telegram bot token"          env-required:"true"                               yaml:"token"`
	Timeout   time.Duration `env:"TELEGRAM_TIMEOUT"    env-default:"10s"                             env-description:"Telegram bot poller timeout"     yaml:"timeout"`
	Chats     []int64       `env:"TELEGRAM_CHATS"      env-description:"Telegram chats to listen to" yaml:"chats"`
	Admins    []int64       `env:"TELEGRAM_ADMINS"     env-description:"Telegram bot admins"         yaml:"admins"`
	Whitelist []int64       `env:"TELEGRAM_WHITELIST"  env-description:"Telegram bot whitelist"      yaml:"whitelist"`
	Blacklist []int64       `env:"TELEGRAM_BLACKLIST"  env-description:"Telegram bot blacklist"      yaml:"blacklist"`
	IgnoreVia bool          `env:"TELEGRAM_IGNORE_VIA" env-default:"false"                           env-description:"Ignore messages from other bots" yaml:"ignore_via"`
}

type CaptchaConfig struct {
	Length     int           `env:"CAPTCHA_LENGTH"     env-default:"6"   env-description:"Captcha length"          yaml:"length"`
	Width      int           `env:"CAPTCHA_WIDTH"      env-default:"480" env-description:"Captcha image width"     yaml:"width"`
	Height     int           `env:"CAPTCHA_HEIGHT"     env-default:"180" env-description:"Captcha image height"    yaml:"height"`
	Expiration time.Duration `env:"CAPTCHA_EXPIRATION" env-default:"10m" env-description:"Captcha expiration time" yaml:"expiration"`
}

// API config.
type APIConfig struct {
	Host         string        `env:"API_HOST"          env-default:"localhost" env-description:"API host address to bind to" yaml:"host"`
	Port         int           `env:"API_PORT"          env-default:"8080"      env-description:"API port to bind to"         yaml:"port"`
	ReadTimeout  time.Duration `env:"API_READ_TIMEOUT"  env-default:"10s"       yaml:"read_timeout"`
	WriteTimeout time.Duration `env:"API_WRITE_TIMEOUT" env-default:"10s"       yaml:"write_timeout"`
	IdleTimeout  time.Duration `env:"API_IDLE_TIMEOUT"  env-default:"15s"       yaml:"idle_timeout"`
}

// SQLite / PostgreSQL / MySQL config for GORM dialector.
type DatabaseConfig struct {
	Driver     string `env:"DATABASE_DRIVER"     env-default:"sqlite3"    env-description:"Database driver to use: sqlite3 | postgres | mysql" yaml:"driver"`
	Connection string `env:"DATABASE_CONNECTION" env-default:"db.sqlite3" env-description:"Connection string or path for SQLite database"      yaml:"connection"`
	Logging    bool   `env:"DATABASE_LOGGING"    env-default:"false"      env-description:"Enable database logging"                                  yaml:"logging"`
}

// IsValid - check if the google sign in config is valid.
func IsValid() bool {
	return true
}

// Error - наша собственная структура для ошибок.
type Error struct {
	Message string
}

// Error - реализация метода Error для нашего типа ошибки.
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
