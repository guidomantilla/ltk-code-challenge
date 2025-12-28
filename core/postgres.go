package core

import (
	"context"
	"fmt"

	pgx "github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type DBInstance interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	tracer trace.Tracer
	pool   DBInstance
}

func NewRepository(pool DBInstance) *Repository {
	return &Repository{
		tracer: otel.GetTracerProvider().Tracer("ltk-code-challenge/core"),
		pool:   pool,
	}
}

func (r *Repository) SaveEvent(ctx context.Context, event *Event) (*Event, error) {
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

func (r *Repository) GetEventById(ctx context.Context, id string) (*Event, error) {
	ctx, span := r.tracer.Start(ctx, "repository.GetEventById")
	defer span.End()

	var e Event

	err := r.pool.QueryRow(
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
