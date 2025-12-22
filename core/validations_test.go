package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEvent(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		event   Event
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid event",
			event: Event{
				Title:     "Valid Title",
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			wantErr: false,
		},
		{
			name: "empty title",
			event: Event{
				Title:     "   ",
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "title too long",
			event: Event{
				Title:     string(make([]byte, 101)),
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			wantErr: true,
			errMsg:  "title is too long (100 characters tops)",
		},
		{
			name: "end time before start time",
			event: Event{
				Title:     "Valid Title",
				StartTime: now,
				EndTime:   now.Add(-time.Hour),
			},
			wantErr: true,
			errMsg:  "end time must be after start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateEvent(tt.event)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
