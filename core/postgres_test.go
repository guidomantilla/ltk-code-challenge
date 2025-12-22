package core

import (
	"context"
	"errors"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_SaveEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name       string
		event      *Event
		mockSetup  func(mock pgxmock.PgxPoolIface)
		wantErr    bool
		wantResult *Event
	}{
		{
			name: "success",
			event: &Event{
				Title:       "Test",
				Description: "Desc",
				StartTime:   now,
				EndTime:     now.Add(time.Hour),
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()

				rows := pgxmock.NewRows([]string{"id", "title", "description", "start_time", "end_time", "created_at"}).
					AddRow("uuid-1", "Test", "Desc", now, now.Add(time.Hour), now)
				mock.ExpectQuery("INSERT INTO events").
					WithArgs("Test", "Desc", now, now.Add(time.Hour)).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
			wantErr: false,
			wantResult: &Event{
				Id:          "uuid-1",
				Title:       "Test",
				Description: "Desc",
				StartTime:   now,
				EndTime:     now.Add(time.Hour),
				CreatedAt:   now,
			},
		},
		{
			name:  "begin failure",
			event: &Event{Title: "Test"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(errors.New("begin error"))
			},
			wantErr: true,
		},
		{
			name:  "commit failure",
			event: &Event{Title: "Test"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()

				rows := pgxmock.NewRows([]string{"id", "title", "description", "start_time", "end_time", "created_at"}).
					AddRow("uuid-1", "Test", "", time.Time{}, time.Time{}, time.Time{})
				mock.ExpectQuery("INSERT INTO events").
					WithArgs("Test", "", time.Time{}, time.Time{}).
					WillReturnRows(rows)
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)

			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewRepository(mock)
			got, err := repo.SaveEvent(ctx, tt.event)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				if tt.wantResult != nil {
					assert.Equal(t, tt.wantResult, got)
				}
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetEventById(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name       string
		id         string
		mockSetup  func(mock pgxmock.PgxPoolIface)
		wantErr    bool
		wantResult *Event
	}{
		{
			name: "success",
			id:   "uuid-1",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "title", "description", "start_time", "end_time", "created_at"}).
					AddRow("uuid-1", "Test", "Desc", now, now.Add(time.Hour), now)
				mock.ExpectQuery("SELECT (.+) FROM events WHERE id = \\$1").
					WithArgs("uuid-1").
					WillReturnRows(rows)
			},
			wantErr: false,
			wantResult: &Event{
				Id:          "uuid-1",
				Title:       "Test",
				Description: "Desc",
				StartTime:   now,
				EndTime:     now.Add(time.Hour),
				CreatedAt:   now,
			},
		},
		{
			name: "not found",
			id:   "uuid-empty",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM events WHERE id = \\$1").
					WithArgs("uuid-empty").
					WillReturnError(errors.New("no rows in result set"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)

			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewRepository(mock)
			got, err := repo.GetEventById(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult, got)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
