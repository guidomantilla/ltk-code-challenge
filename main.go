package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qmdx00/lifecycle"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"ltk-code-challenge/core"
	"ltk-code-challenge/pkg/resources"
	"ltk-code-challenge/pkg/servers"
)

func main() {
	var err error

	ctx := context.Background()
	name, version := "ltk-code-challenge", "1.0"

	viper.AutomaticEnv()

	stopFn, err := resources.Trace(ctx)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to setup tracing: %v", err))
	}
	defer stopFn(ctx)

	stopFn, err = resources.Profile(ctx)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to setup profiling: %v", err))
	}
	defer stopFn(ctx)

	stopFn, err = resources.Measure(ctx)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to setup metrics: %v", err))
	}
	defer stopFn(ctx)

	stopFn, err = resources.Logger(ctx)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to setup logging: %v", err))
	}
	defer stopFn(ctx)

	pool, err := resources.CreateDatabaseConnectionPool(ctx)
	if err != nil {
		//nolint:gocritic
		log.Fatal().Msg(fmt.Sprintf("Unable to create database connection pool: %v", err))
	}

	repo := core.NewRepository(pool)
	handlers := core.NewHandlers(repo)

	app := lifecycle.NewApp(
		lifecycle.WithName(name),
		lifecycle.WithVersion(version),
		lifecycle.WithSignal(syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL),
	)
	{

		tracerMiddleware := otelgin.Middleware("ltk-code-challenge")
		metricsMiddleware := NewHTTPMetrics().Middleware()

		router := gin.Default()
		router.Use(tracerMiddleware)
		router.Use(metricsMiddleware)

		router.POST("/events", handlers.PostEvents)
		router.GET("/events/:id", handlers.GetEvents)

		httpServer := &http.Server{
			Addr:              net.JoinHostPort("localhost", "8080"),
			Handler:           router,
			ReadHeaderTimeout: 60000,
		}

		app.Attach(servers.BuildHttpServer(httpServer))
		app.Attach(servers.BuildBaseServer(pool))
	}

	if app.Run() != nil {
		log.Err(err).Msg("application encountered an error")
	}
}

type HTTPMetrics struct {
	reqs    metric.Int64Counter
	latency metric.Float64Histogram
}

func NewHTTPMetrics() *HTTPMetrics {
	meter := otel.Meter("ltk-code-challenge/http")

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
