package metrics

// metricsLoggerFake is a no-op implementation of MetricsLogger
type metricsLoggerFake struct{}

// Ensure FakeMetricsLogger implements MetricsLoggerInterface
var _ MetricsLogger = (*metricsLoggerFake)(nil)

// NewFakeMetrics creates an instance of FakeMetricsLogger
func NewMetricsFake() MetricsLogger {
	return &metricsLoggerFake{}
}

// LogEvent is a no-op method for FakeMetricsLogger
func (this *metricsLoggerFake) LogEvent(eventName string, tags map[string]string, fields map[string]interface{}) {
	// No operation, this is a fake logger
}

// LogChatEvent is a no-op method for FakeMetricsLogger
func (this *metricsLoggerFake) LogChatEvent(eventName string, chatID int64, fields map[string]interface{}) {
	// No operation, this is a fake logger
}

// Close is a no-op method for FakeMetricsLogger
func (this *metricsLoggerFake) Close() {
	// No operation, this is a fake logger
}
