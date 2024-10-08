package storagelogger

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// GormSlogLogger is a custom GORM logger that uses slog.Logger for logging.
type GormSlogLogger struct {
	logger *slog.Logger
	level  logger.LogLevel
}

// NewGormSlogLogger creates a new GormSlogLogger instance.
func NewGormSlogLogger(slog *slog.Logger) *GormSlogLogger {
	return &GormSlogLogger{
		logger: slog,
	}
}

// LogMode sets the log level for GORM logger.
func (l *GormSlogLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.level = level

	return l
}

// Info logs info-level messages.
func (l *GormSlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		l.logger.InfoContext(ctx, msg, slog.Any("data", data))
	}
}

// Warn logs warning-level messages.
func (l *GormSlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		l.logger.WarnContext(ctx, msg, slog.Any("data", data))
	}
}

// Error logs error-level messages.
func (l *GormSlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		l.logger.ErrorContext(ctx, msg, slog.Any("data", data))
	}
}

// Trace logs SQL queries with their execution time, affected rows, and errors.
func (l *GormSlogLogger) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil {
		l.logger.ErrorContext(ctx, "SQL execution error",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
			slog.Any("err", err),
		)
	} else {
		l.logger.InfoContext(ctx, "SQL executed",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	}
}
