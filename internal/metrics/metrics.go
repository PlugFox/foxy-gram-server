package metrics

import (
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// Metrics defines the contract for logging metrics
type Metrics interface {
	LogEvent(eventName string, tags map[string]string, fields map[string]interface{})
	LogChatEvent(eventName string, chatID int64, fields map[string]interface{})
	Close()
}

type metricsImpl struct {
	client      influxdb2.Client
	writeAPI    api.WriteAPI
	org         string
	bucket      string
	defaultTags map[string]string // Constant tags, like bot ID
}

// Ensure MetricsLogger implements MetricsLoggerInterface
var _ Metrics = (*metricsImpl)(nil)

// New initializes the logger with constant tags like bot ID
func NewMetricsImpl(url string, token string, org string, bucket string, defaultTags map[string]string) Metrics {
	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPI(org, bucket)
	return &metricsImpl{
		client:      client,
		writeAPI:    writeAPI,
		org:         org,
		bucket:      bucket,
		defaultTags: defaultTags,
	}
}

// Universal method to log an event with customizable tags and fields
func (metrics *metricsImpl) LogEvent(eventName string, tags map[string]string, fields map[string]interface{}) {
	if len(fields) == 0 {
		return
	}

	point := influxdb2.NewPointWithMeasurement("bot_event").
		AddTag("event", eventName).
		SetTime(time.Now())

	// Add constant default tags
	for key, value := range metrics.defaultTags {
		point.AddTag(key, value)
	}

	// Add custom tags
	for key, value := range tags {
		point.AddTag(key, value)
	}

	// Add custom fields
	for key, value := range fields {
		point.AddField(key, value)
	}

	metrics.writeAPI.WritePoint(point)
	// fmt.Printf("Logged event: %s with tags %v and fields %v\n", eventName, tags, fields)
}

// Specific method for logging chat-related events
func (metrics *metricsImpl) LogChatEvent(eventName string, chatID int64, fields map[string]interface{}) {
	if chatID == 0 {
		return
	}

	tags := map[string]string{
		"chat_id": strconv.FormatInt(chatID, 10),
	}

	metrics.LogEvent(eventName, tags, fields)
}

// Close flushes the write API and closes the client
func (metrics *metricsImpl) Close() {
	metrics.writeAPI.Flush()
	metrics.client.Close()
}
