package log

import (
	"io"
)

type Config struct {
	writer    io.Writer
	addSource bool
	level     string
}

type Option interface {
	apply(config *Config)
}

type OptionFunc func(*Config)

func (o OptionFunc) apply(c *Config) {
	o(c)
}

func WithWriter(writer io.Writer) OptionFunc {
	if writer == nil {
		return func(*Config) {}
	}

	return func(config *Config) {
		config.writer = writer
	}
}

// Levels:
// - DEBUG
// - INFO
// - WARN
// - ERROR.
func WithLevel(level string) OptionFunc {
	if level == "" {
		return func(*Config) {}
	}

	return func(config *Config) {
		config.level = level
	}
}

func WithSource() OptionFunc {
	return func(config *Config) {
		config.addSource = true
	}
}
