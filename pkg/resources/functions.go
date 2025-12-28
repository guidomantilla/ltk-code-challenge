package resources

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func CreateTracer(ctx context.Context) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tp, err := newTracerProvider(ctx)
	if err != nil {
		return func(context.Context) error { return nil }, fmt.Errorf("failed to create tracer provider: %w", err)
	}
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func newTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Si tu app corre en Docker (misma red del compose), cambia a "otel-collector:4317"
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the OTLP exporter: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
	), nil
}

func CreateDatabaseConnectionPool(ctx context.Context) (*pgxpool.Pool, error) {
	//nolint:nosprintfhostport
	cfg, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"), viper.GetString("DB_PORT"), viper.GetString("DB_NAME")))
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Unable to parse database connection string: %v", err))
		return nil, fmt.Errorf("failed to parse database connection string: %w", err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Unable to connect to database: %v", err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Unable to ping to database: %v", err))
		return nil, fmt.Errorf("failed to ping to database: %w", err)
	}

	return pool, nil
}
