package core

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"ltk-code-challenge/pkg/resources"
)

type Repository interface {
	SaveEvent(ctx context.Context, event *Event) (*Event, error)
	GetEventById(ctx context.Context, id string) (*Event, error)
}

type repository struct {
	tracer  trace.Tracer
	metrics *DBMetrics
	pool    resources.DBInstance
}

func NewRepository(pool resources.DBInstance) Repository {
	return &repository{
		tracer:  otel.GetTracerProvider().Tracer("ltk-code-challenge/core"),
		metrics: NewDBMetrics(),
		pool:    pool,
	}
}

func (r *repository) SaveEvent(ctx context.Context, event *Event) (*Event, error) {
	start := time.Now()

	var err error

	defer func() { r.metrics.Observe(ctx, "save_event", start, err) }()

	ctx, span := r.tracer.Start(ctx, "repository.SaveEvent")
	defer span.End()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var savedEvent Event

	_ = tx.QueryRow(ctx,
		"INSERT INTO events (title, description, start_time, end_time) "+
			"VALUES ($1, $2, $3, $4) "+
			"RETURNING id, title, description, start_time, end_time, created_at",
		event.Title, event.Description, event.StartTime, event.EndTime).
		Scan(&savedEvent.Id, &savedEvent.Title, &savedEvent.Description, &savedEvent.StartTime, &savedEvent.EndTime, &savedEvent.CreatedAt)

	err = tx.Commit(ctx)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &savedEvent, nil
}

func (r *repository) GetEventById(ctx context.Context, id string) (*Event, error) {
	start := time.Now()

	var err error

	defer func() { r.metrics.Observe(ctx, "get_event_by_id", start, err) }()

	ctx, span := r.tracer.Start(ctx, "repository.GetEventById")
	defer span.End()

	var e Event

	err = r.pool.QueryRow(
		ctx,
		`SELECT id, title, description, start_time, end_time, created_at
		 FROM events
		 WHERE id = $1`,
		id,
	).Scan(
		&e.Id,
		&e.Title,
		&e.Description,
		&e.StartTime,
		&e.EndTime,
		&e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get event by id: %w", err)
	}

	return &e, nil
}

/*

 */

type DBMetrics struct {
	qTotal   metric.Int64Counter
	qErrors  metric.Int64Counter
	qLatency metric.Float64Histogram
}

func NewDBMetrics() *DBMetrics {
	meter := otel.Meter("ltk-code-challenge/db")

	qTotal, _ := meter.Int64Counter("db.query.total")
	qErrors, _ := meter.Int64Counter("db.query.errors.total")
	qLatency, _ := meter.Float64Histogram("db.query.duration.ms")

	return &DBMetrics{qTotal: qTotal, qErrors: qErrors, qLatency: qLatency}
}

func (m *DBMetrics) Observe(ctx context.Context, op string, start time.Time, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "postgres"),
		attribute.String("db.operation", op), // ej: "save_event", "get_event"
	}

	m.qTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	ms := float64(time.Since(start).Milliseconds())
	m.qLatency.Record(ctx, ms, metric.WithAttributes(attrs...))

	if err != nil {
		m.qErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}
