package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/guidomantilla/yarumo/managed"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func noopStop(_ context.Context, _ time.Duration) {}

func CreateDatabaseConnectionPool(ctx context.Context) (*pgxpool.Pool, managed.StopFn, error) {
	//nolint:nosprintfhostport
	cfg, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"), viper.GetString("DB_PORT"), viper.GetString("DB_NAME")))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg(fmt.Sprintf("Unable to parse database connection string: %v", err))
		return nil, noopStop, fmt.Errorf("failed to parse database connection string: %w", err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg(fmt.Sprintf("Unable to connect to database: %v", err))
		return nil, noopStop, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg(fmt.Sprintf("Unable to ping to database: %v", err))
		return nil, noopStop, fmt.Errorf("failed to ping to database: %w", err)
	}

	stopFn := func(ctx context.Context, timeout time.Duration) {
		pool.Close()
	}

	return pool, stopFn, nil
}
