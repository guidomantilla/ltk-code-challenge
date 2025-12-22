package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock of the RepositoryInterface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveEvent(ctx context.Context, event *Event) (*Event, error) {
	args := m.Called(ctx, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*Event), args.Error(1)
}

func (m *MockRepository) GetEventById(ctx context.Context, id string) (*Event, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*Event), args.Error(1)
}

func TestHandlers_PostEvents(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name           string
		body           any
		mockReturn     *Event
		mockErr        error
		expectedStatus int
	}{
		{
			name: "success",
			body: Event{
				Title:     "Test Event",
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			mockReturn: &Event{
				Id:        "uuid-123",
				Title:     "Test Event",
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			mockErr:        nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "validation failure",
			body: Event{
				Title: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "repository failure",
			body: Event{
				Title:     "Test Event",
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			mockReturn:     nil,
			mockErr:        errors.New("db error"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			body:           "invalid",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)
			if tt.name == "success" || tt.name == "repository failure" {
				mockRepo.On("SaveEvent", mock.Anything, mock.Anything).Return(tt.mockReturn, tt.mockErr)
			}

			h := NewHandlers(mockRepo)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var jsonBody []byte
			if s, ok := tt.body.(string); ok {
				jsonBody = []byte(s)
			} else {
				jsonBody, _ = json.Marshal(tt.body)
			}

			c.Request = httptest.NewRequest(http.MethodPost, "/events", bytes.NewBuffer(jsonBody))

			h.PostEvents(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestHandlers_GetEvents(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		idParam        string
		reqBody        any
		mockReturn     *Event
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "success",
			idParam:        "123",
			reqBody:        "",
			mockReturn:     &Event{Id: "123", Title: "Event"},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			idParam:        "456",
			reqBody:        "",
			mockReturn:     nil,
			mockErr:        ErrEventNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "non empty body",
			idParam:        "123",
			reqBody:        "something",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing id",
			idParam:        "",
			reqBody:        "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "repository error",
			idParam:        "123",
			reqBody:        "",
			mockReturn:     nil,
			mockErr:        errors.New("db error"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "read body error",
			idParam:        "123",
			reqBody:        &errReader{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)
			if tt.name == "success" || tt.name == "not found" || tt.name == "repository error" {
				mockRepo.On("GetEventById", mock.Anything, tt.idParam).Return(tt.mockReturn, tt.mockErr)
			}

			h := NewHandlers(mockRepo)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = []gin.Param{{Key: "id", Value: tt.idParam}}

			var reader io.Reader
			if r, ok := tt.reqBody.(io.Reader); ok {
				reader = r
			} else if s, ok := tt.reqBody.(string); ok {
				reader = bytes.NewBufferString(s)
			} else {
				reader = bytes.NewBufferString("")
			}

			c.Request = httptest.NewRequest(http.MethodGet, "/events/"+tt.idParam, reader)

			h.GetEvents(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}
