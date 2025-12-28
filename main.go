package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/qmdx00/lifecycle"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"ltk-code-challenge/core"
	"ltk-code-challenge/pkg/resources"
	"ltk-code-challenge/pkg/servers"
)

func main() {
	var err error

	ctx := context.Background()
	name, version := "ltk-code-challenge", "1.0"

	viper.AutomaticEnv()

	otelShutdown, err := resources.CreateTracer(ctx)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to setup OpenTelemetry SDK: %v", err))
	}

	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

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
		router := gin.Default()
		router.Use(otelgin.Middleware("ltk-code-challenge"))

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
