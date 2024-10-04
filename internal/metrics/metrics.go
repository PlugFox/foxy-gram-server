package metrics

import (
	"fmt"
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// MetricsLogger defines the contract for logging metrics
type MetricsLogger interface {
	LogEvent(eventName string, tags map[string]string, fields map[string]interface{})
	LogChatEvent(eventName string, chatID int64, fields map[string]interface{})
	Close()
}

type metricsLoggerImpl struct {
	client      influxdb2.Client
	writeAPI    api.WriteAPI
	org         string
	bucket      string
	defaultTags map[string]string // Constant tags, like bot ID
}

// Ensure MetricsLogger implements MetricsLoggerInterface
var _ MetricsLogger = (*metricsLoggerImpl)(nil)

// New initializes the logger with constant tags like bot ID
func NewMetricsImpl(url string, token string, org string, bucket string, defaultTags map[string]string) MetricsLogger {
	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPI(org, bucket)
	return &metricsLoggerImpl{
		client:      client,
		writeAPI:    writeAPI,
		org:         org,
		bucket:      bucket,
		defaultTags: defaultTags,
	}
}

// Universal method to log an event with customizable tags and fields
func (this *metricsLoggerImpl) LogEvent(eventName string, tags map[string]string, fields map[string]interface{}) {
	if fields == nil || len(fields) == 0 {
		return
	}

	point := influxdb2.NewPointWithMeasurement("bot_event").
		AddTag("event", eventName).
		SetTime(time.Now())

	// Add constant default tags
	for key, value := range this.defaultTags {
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

	this.writeAPI.WritePoint(point)
	fmt.Printf("Logged event: %s with tags %v and fields %v\n", eventName, tags, fields)
}

// Specific method for logging chat-related events
func (this *metricsLoggerImpl) LogChatEvent(eventName string, chatID int64, fields map[string]interface{}) {
	if chatID == 0 {
		return
	}

	tags := map[string]string{
		"chat_id": strconv.FormatInt(chatID, 10),
	}

	this.LogEvent(eventName, tags, fields)
}

// Close flushes the write API and closes the client
func (this *metricsLoggerImpl) Close() {
	this.writeAPI.Flush()
	this.client.Close()
}
