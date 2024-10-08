package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var errorFailedToReadConfig = fmt.Errorf("failed to read config")

// Config is the main config struct.
type Config struct {
	Environment string `env:"ENVIRONMENT" env-default:"production" env-description:"Environment name"                                     yaml:"environment"`
	Secret      string `env:"SECRET"      env-default:""           env-description:"Secret key for JWT token signing and validation"      yaml:"secret"`
	Verbose     string `env:"VERBOSE"     env-default:"warn"       env-description:"Verbose mode for output: debug | info | warn | error" yaml:"verbose"`

	Proxy    ProxyConfig    `env-description:"Proxy SOCKS5 server config" yaml:"proxy"`
	Metrics  MetricsConfig  `yaml:"metrics"`
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

// Metrics config.
type MetricsConfig struct {
	URL    string `env:"METRICS_URL"    env-description:"Metrics URL"    yaml:"url"`
	Token  string `env:"METRICS_TOKEN"  env-description:"Metrics token"  yaml:"token"`
	Org    string `env:"METRICS_ORG"    env-description:"Metrics org"    yaml:"org"`
	Bucket string `env:"METRICS_BUCKET" env-description:"Metrics bucket" yaml:"bucket"`
}

// IsValid - check if the metrics config is valid.
func (config *MetricsConfig) IsValid() bool {
	return config != nil && config.URL != "" && config.Token != "" && config.Org != "" && config.Bucket != ""
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

// Captcha config.
type CaptchaConfig struct {
	Length     int           `env:"CAPTCHA_LENGTH"     env-default:"6"   env-description:"Captcha length"          yaml:"length"`
	Width      int           `env:"CAPTCHA_WIDTH"      env-default:"480" env-description:"Captcha image width"     yaml:"width"`
	Height     int           `env:"CAPTCHA_HEIGHT"     env-default:"180" env-description:"Captcha image height"    yaml:"height"`
	Expiration time.Duration `env:"CAPTCHA_EXPIRATION" env-default:"10m" env-description:"Captcha expiration time" yaml:"expiration"`
}

// API config.
type APIConfig struct {
	Host         string        `env:"API_HOST"          env-default:""     env-description:"API host address to bind to" yaml:"host"`
	Port         int           `env:"API_PORT"          env-default:"8080" env-description:"API port to bind to"         yaml:"port"`
	Timeout      time.Duration `env:"API_TIMEOUT"       env-default:"15s"  yaml:"timeout"`
	ReadTimeout  time.Duration `env:"API_READ_TIMEOUT"  env-default:"10s"  yaml:"read_timeout"`
	WriteTimeout time.Duration `env:"API_WRITE_TIMEOUT" env-default:"10s"  yaml:"write_timeout"`
	IdleTimeout  time.Duration `env:"API_IDLE_TIMEOUT"  env-default:"15s"  yaml:"idle_timeout"`
}

// SQLite / PostgreSQL / MySQL config for GORM dialector.
type DatabaseConfig struct {
	Driver     string `env:"DATABASE_DRIVER"     env-default:"sqlite3"    env-description:"Database driver to use: sqlite3 | postgres | mysql" yaml:"driver"`
	Connection string `env:"DATABASE_CONNECTION" env-default:"db.sqlite3" env-description:"Connection string or path for SQLite database"      yaml:"connection"`
	Logging    bool   `env:"DATABASE_LOGGING"    env-default:"false"      env-description:"Enable database logging"                            yaml:"logging"`
}

// MustLoadConfig - load config from file or environment variables.
func MustLoadConfig() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yml"
	}

	var config Config

	// Check if config file exists
	_, err := os.Stat(configPath)

	if os.IsNotExist(err) {
		// Read environment variables if config file does not exist
		err = cleanenv.ReadEnv(&config)
	} else if err == nil {
		// Read config file if it exists
		err = cleanenv.ReadConfig(configPath, &config)
	}

	// Check if there was an error reading the config file
	if err != nil {
		return nil, errorFailedToReadConfig
	}

	return &config, nil
}
