package global

import (
	slog "log/slog"

	conf "github.com/plugfox/foxy-gram-server/internal/config"
)

var (
	Logger *slog.Logger //nolint:gochecknoglobals
	Config *conf.Config //nolint:gochecknoglobals
)
