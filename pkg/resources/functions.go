package resources

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func CreateDatabaseConnectionPool(ctx context.Context) (*pgxpool.Pool, error) {

	//nolint:nosprintfhostport
	pool, err := pgxpool.New(ctx, fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"), viper.GetString("DB_PORT"), viper.GetString("DB_NAME")))
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

func SetupOTelSDK(ctx context.Context) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(newPropagator())

	tp, err := newTracerProvider(ctx)
	if err != nil {
		return func(context.Context) error { return nil }, err
	}
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Si tu app corre en Docker (misma red del compose), cambia a "otel-collector:4317"
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
	), nil
}
