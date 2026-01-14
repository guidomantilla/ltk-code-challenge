package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guidomantilla/yarumo/common/diagnostics"
	commonhttp "github.com/guidomantilla/yarumo/common/http"
	"github.com/guidomantilla/yarumo/config"
	"github.com/guidomantilla/yarumo/managed"
	telemetry "github.com/guidomantilla/yarumo/telemetry/otel"
	"github.com/rs/zerolog/log"

	"ltk-code-challenge/core"
	"ltk-code-challenge/pkg/resources"
)

func main() {
	var err error

	name, version, env := "ltk-code-challenge", "1.0", "local"

	// 1. Config (Logger base included)
	ctx := config.Default(context.Background(), name, version, env)
	startupLogger := log.Ctx(ctx).With().Str("stage", "startup").Str("component", "main").Logger()
	shutdownLogger := log.Ctx(ctx).With().Str("stage", "shut down").Str("component", "main").Logger()

	startupLogger.Info().Msg("application starting up")
	defer shutdownLogger.Info().Msg("application stopped")

	hookFn := func(ctx context.Context) (context.Context, error) {
		log.Logger = log.Logger.Hook(resources.NewZerologHook(name, version))
		return log.Logger.WithContext(ctx), nil
	}

	// 3. Telemetry (traces/metrics/logs)
	// 4. Bridge zerolog -> OTel Logs (still prints to stdout; additionally exports via OTLP to the collector)
	ctx, stopFn, err := telemetry.Observe(ctx, name, version, env, hookFn, telemetry.WithInsecure())
	if err != nil {
		shutdownLogger.Fatal().Err(err).Msg(fmt.Sprintf("unable to setup otel telemetry: %v", err))
	}
	defer stopFn(ctx, 15*time.Second)

	// 5. Recursos “core” (dependencies de negocio)
	pool, stopFn, err := resources.CreateDatabaseConnectionPool(ctx)
	if err != nil {
		shutdownLogger.Fatal().Err(err).Msg(fmt.Sprintf("unable to create database connection pool: %v", err))
	}
	defer stopFn(ctx, 15*time.Second)

	// 6. Wiring
	repo := core.NewRepository(pool)
	handlers := core.NewHandlers(repo)

	// 7. Daemons/servers setup

	gin.SetMode(gin.ReleaseMode)

	restHandler := gin.Default()
	restHandler.Use(resources.TracerMiddleware(name))
	restHandler.Use(resources.MeterMiddleware(name))

	restHandler.POST("/events", handlers.PostEvents)
	restHandler.GET("/events/:id", handlers.GetEvents)

	// 8. Daemons/servers lifecycle

	errChan := make(chan error, 16)

	_, stopFn, err = managed.BuildBaseWorker(ctx, "keep-alive-worker", nil, errChan)
	if err != nil {
		shutdownLogger.Fatal().Err(err).Msg("unable to build base server")
	}
	defer stopFn(ctx, 15*time.Second)

	_, stopFn, err = managed.BuildTraceFlightRecorderWorker(ctx, "trace-flight-recorder", diagnostics.NewTraceFlightRecorder(), errChan)
	if err != nil {
		shutdownLogger.Fatal().Err(err).Msg("unable to build trace flight recorder worker")
	}
	defer stopFn(ctx, 15*time.Second)

	_, stopFn, err = managed.BuildHttpServer(ctx, "debug-server", commonhttp.NewServer("localhost", "6060", diagnostics.NewPprofHandler()), errChan)
	if err != nil {
		shutdownLogger.Fatal().Err(err).Str("component", "main").Msg("unable to build debug server")
	}
	defer stopFn(ctx, 15*time.Second)

	_, stopFn, err = managed.BuildHttpServer(ctx, "rest-server", commonhttp.NewServer("localhost", "8080", restHandler), errChan)
	if err != nil {
		shutdownLogger.Fatal().Err(err).Str("component", "main").Msg("unable to build rest server")
	}
	defer stopFn(ctx, 15*time.Second)

	startupLogger.Info().Msg("application running")

	// 9. Wait for shutdown signal

	notifyCtx, cancelNotifyFn := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancelNotifyFn()

	select {
	case <-notifyCtx.Done():
		startupLogger.Info().Msg("application shutdown requested")
	case runErr := <-errChan:
		shutdownLogger.Error().Err(runErr).Msg("runtime error")
	}
}
