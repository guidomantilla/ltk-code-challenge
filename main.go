package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guidomantilla/yarumo/managed"
	telemetry "github.com/guidomantilla/yarumo/telemetry/otel"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

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

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(resources.TracerMiddleware(name))
	router.Use(resources.MeterMiddleware(name))

	router.POST("/events", handlers.PostEvents)
	router.GET("/events/:id", handlers.GetEvents)

	httpServer := &http.Server{
		Addr:              net.JoinHostPort("localhost", "8080"),
		Handler:           router,
		ReadHeaderTimeout: 60 * time.Second,
	}

	errChan := make(chan error, 16)
	/*
		_, stopFn, err = managed.BuildBaseServer(ctx, "base-server", errChan)
		if err != nil {
			log.Ctx(ctx).Fatal().Err(err).Str("stage", "shut down").Str("component", "main").Msg("unable to build base server")
		}
		defer stopFn(ctx, 15*time.Second)
	*/
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
