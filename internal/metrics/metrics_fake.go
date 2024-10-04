package metrics

// metricsFake is a no-op implementation of MetricsLogger
type metricsFake struct{}

// Ensure FakeMetricsLogger implements MetricsLoggerInterface
var _ Metrics = (*metricsFake)(nil)

// NewFakeMetrics creates an instance of FakeMetricsLogger
func NewMetricsFake() Metrics {
	return &metricsFake{}
}

// LogEvent is a no-op method for FakeMetricsLogger
func (metrics *metricsFake) LogEvent(_ string, _ map[string]string, _ map[string]interface{}) {
	// No operation, this is a fake logger
}

// LogChatEvent is a no-op method for FakeMetricsLogger
func (metrics *metricsFake) LogChatEvent(_ string, _ int64, _ map[string]interface{}) {
	// No operation, this is a fake logger
}

// Close is a no-op method for FakeMetricsLogger
func (metrics *metricsFake) Close() {
	// No operation, this is a fake logger
}
