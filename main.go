package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guidomantilla/yarumo/managed"
	telemetry "github.com/guidomantilla/yarumo/telemetry/otel"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"ltk-code-challenge/core"
	"ltk-code-challenge/pkg/resources"
)

func main() {
	var err error

	name, version, env := "ltk-code-challenge", "1.0", "local"

	// 1. Config
	viper.AutomaticEnv()

	// 2. Logger base
	ctx := log.Logger.WithContext(context.Background())

	log.Ctx(ctx).Info().Str("stage", "startup").Str("component", "main").Msg("application starting up")
	defer log.Ctx(ctx).Info().Str("stage", "shut down").Str("component", "main").Msg("application stopped")

	hookFn := func(ctx context.Context) (context.Context, error) {
		log.Logger = log.Logger.Hook(resources.NewZerologHook(name, version))
		return log.Logger.WithContext(ctx), nil
	}

	// 3. Telemetry (traces/metrics/logs)
	// 4. Bridge zerolog -> OTel Logs (still prints to stdout; additionally exports via OTLP to the collector)
	ctx, stopFn, err := telemetry.Observe(ctx, name, version, env, hookFn, telemetry.WithInsecure())
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Str("stage", "shut down").Str("component", "main").Msg(fmt.Sprintf("unable to setup otel telemetry: %v", err))
	}
	defer stopFn(ctx, 15*time.Second)

	// 5. Recursos “core” (dependencies de negocio)
	pool, stopFn, err := resources.CreateDatabaseConnectionPool(ctx)
	if err != nil {

		log.Ctx(ctx).Fatal().Err(err).Str("stage", "shut down").Str("component", "main").Msg(fmt.Sprintf("unable to create database connection pool: %v", err))
	}
	defer stopFn(ctx, 15*time.Second)

	// 6. Wiring
	repo := core.NewRepository(pool)
	handlers := core.NewHandlers(repo)

	// 7. Daemons/servers
	tracerMiddleware := otelgin.Middleware("ltk-code-challenge")
	metricsMiddleware := NewHTTPMetrics().Middleware()

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(tracerMiddleware)
	router.Use(metricsMiddleware)

	router.POST("/events", handlers.PostEvents)
	router.GET("/events/:id", handlers.GetEvents)

	httpServer := &http.Server{
		Addr:              net.JoinHostPort("localhost", "8080"),
		Handler:           router,
		ReadHeaderTimeout: 60 * time.Second,
	}

	errChan := make(chan error, 16)

	_, stopFn, err = managed.BuildBaseServer(ctx, "base-server", errChan)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Str("stage", "shut down").Str("component", "main").Msg("unable to build base server")
	}
	defer stopFn(ctx, 15*time.Second)

	_, stopFn, err = managed.BuildHttpServer(ctx, "http-server", httpServer, errChan)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Str("stage", "shut down").Str("component", "main").Msg("unable to build http server")
	}
	defer stopFn(ctx, 15*time.Second)

	log.Ctx(ctx).Info().Str("stage", "startup").Str("component", "main").Msg("application running")

	notifyCtx, cancelNotifyFn := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancelNotifyFn()

	select {
	case <-notifyCtx.Done():
		log.Ctx(ctx).Info().Str("stage", "shut down").Str("component", "main").Msg("application shutdown requested")
	case runErr := <-errChan:
		log.Ctx(ctx).Error().Str("stage", "shut down").Str("component", "main").Err(runErr).Msg("runtime error")
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
