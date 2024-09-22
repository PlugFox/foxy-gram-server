package log

import (
	"log"
	"log/slog"
)

type logAdapter struct {
	slog *slog.Logger
}

func NewLogAdapter(logger *slog.Logger) *log.Logger {
	return log.New(&logAdapter{slog: logger}, "", 0)
}

func (a *logAdapter) Write(p []byte) (n int, err error) {
	a.slog.Info(string(p))

	return len(p), nil
}


func (a *logAdapter) Print(p []byte) (n int, err error) {
	a.slog.Info(string(p))

	return len(p), nil
}
