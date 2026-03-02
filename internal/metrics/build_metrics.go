package metrics

import (
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

// BuildMetrics emits DogStatsD metrics for the build pipeline.
type BuildMetrics struct {
	client statsd.ClientInterface
}

// NewBuildMetrics creates a BuildMetrics.
func NewBuildMetrics(client statsd.ClientInterface) *BuildMetrics {
	return &BuildMetrics{client: client}
}

// BuildDuration emits build.duration histogram.
func (m *BuildMetrics) BuildDuration(project, language, status string, d time.Duration) {
	tags := []string{
		"project:" + project,
		"language:" + language,
		"status:" + status,
	}
	_ = m.client.Histogram("build.duration", d.Seconds(), tags, 1)
}

// BuildStatus increments build.status count.
func (m *BuildMetrics) BuildStatus(project, status string) {
	tags := []string{"project:" + project, "status:" + status}
	_ = m.client.Incr("build.status", tags, 1)
}

// QueueWaitTime emits build.queue_wait_time histogram using the published_at timestamp.
func (m *BuildMetrics) QueueWaitTime(publishedAt time.Time) {
	wait := time.Since(publishedAt)
	_ = m.client.Histogram("build.queue_wait_time", wait.Seconds(), nil, 1)
}

// ProjectsAffected emits build.projects_affected gauge.
func (m *BuildMetrics) ProjectsAffected(count int) {
	_ = m.client.Gauge("build.projects_affected", float64(count), nil, 1)
}

// RetryCount increments build.retry_count.
func (m *BuildMetrics) RetryCount(project string, attempt int) {
	tags := []string{
		"project:" + project,
		fmt.Sprintf("attempt:%d", attempt),
	}
	_ = m.client.Incr("build.retry_count", tags, 1)
}
