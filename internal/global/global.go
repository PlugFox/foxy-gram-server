package global

import (
	slog "log/slog"

	conf "github.com/plugfox/foxy-gram-server/internal/config"

	metr "github.com/plugfox/foxy-gram-server/internal/metrics"
)

var (
	Logger  *slog.Logger       //nolint:gochecknoglobals
	Config  *conf.Config       //nolint:gochecknoglobals
	Metrics metr.MetricsLogger //nolint:gochecknoglobals
)
