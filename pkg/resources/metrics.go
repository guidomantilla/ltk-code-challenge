package resources

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HTTPMetrics struct {
	reqs    metric.Int64Counter
	latency metric.Float64Histogram
}

func NewHTTPMetrics(name string) *HTTPMetrics {
	meter := otel.Meter(name)

	reqs, _ := meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("HTTP requests"),
	)
	latency, _ := meter.Float64Histogram(
		"http.server.duration.ms",
		metric.WithDescription("HTTP request duration in milliseconds"),
	)

	return &HTTPMetrics{reqs: reqs, latency: latency}
}

func (m *HTTPMetrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		status := c.Writer.Status()
		method := c.Request.Method

		attrs := []attribute.KeyValue{
			attribute.String("http.route", route),
			attribute.String("http.method", method),
			attribute.Int("http.status_code", status),
			attribute.String("http.status_class", strconv.Itoa(status/100)+"xx"),
		}

		m.reqs.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))
		m.latency.Record(
			c.Request.Context(),
			float64(time.Since(start).Milliseconds()),
			metric.WithAttributes(attrs...),
		)
	}
}
