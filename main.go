package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qmdx00/lifecycle"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"ltk-code-challenge/core"
	"ltk-code-challenge/pkg/servers"
)

func main() {
	var err error

	viper.AutomaticEnv()

	ctx := context.Background()
	app := lifecycle.NewApp(
		lifecycle.WithName("ltk-code-challenge"),
		lifecycle.WithVersion("1.0"),
		lifecycle.WithSignal(syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL),
	)

	var pool *pgxpool.Pool
	{
		//nolint:nosprintfhostport
		pool, err = pgxpool.New(ctx, fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
			viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
			viper.GetString("DB_HOST"), viper.GetString("DB_PORT"), viper.GetString("DB_NAME")))
		if err != nil {
			log.Fatal().Msg(fmt.Sprintf("Unable to connect to database: %v", err))
		}

		err = pool.Ping(ctx)
		if err != nil {
			log.Fatal().Msg(fmt.Sprintf("Unable to ping to database: %v", err))
		}

		log.Info().Msg("Connected to database successfully")
	}

	repo := core.NewRepository(pool)
	handlers := core.NewHandlers(repo)

	{
		router := gin.Default()

		router.POST("/events", handlers.PostEvents)
		router.GET("/events/:id", handlers.GetEvents)

		httpServer := &http.Server{
			Addr:              net.JoinHostPort("localhost", "8080"),
			Handler:           router,
			ReadHeaderTimeout: 60000,
		}

		app.Attach(servers.BuildHttpServer(httpServer))
	}

	if app.Run() != nil {
		log.Err(err).Msg("application encountered an error")
	}
}
