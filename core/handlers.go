package core

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Handlers interface {
	PostEvents(gctx *gin.Context)
	GetEvents(gctx *gin.Context)
}

type handlers struct {
	repository Repository
}

func NewHandlers(repository Repository) Handlers {
	return &handlers{repository: repository}
}

func (h *handlers) PostEvents(gctx *gin.Context) {
	ctx := gctx.Request.Context()

	var event Event

	// Accepts a JSON payload with title, description, start_time, and end_time.
	err := gctx.ShouldBindJSON(&event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to bind JSON")
		gctx.AbortWithStatusJSON(http.StatusInternalServerError, NewError("failed to bind JSON", err))

		return
	}

	// Validates that title is non-empty and <= 100 characters, start_time is before end_time.
	err = ValidateEvent(event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("event validation failed")
		gctx.AbortWithStatusJSON(http.StatusBadRequest, NewError("event validation failed", err))

		return
	}

	// Inserts the event into a PostgreSQL database, generating a UUID for id and setting created_at to current time.
	savedEvent, err := h.repository.SaveEvent(ctx, &event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("saving event failed")
		gctx.AbortWithStatusJSON(http.StatusBadRequest, NewError("saving event failed", err))

		return
	}

	// Returns the created event as JSON with HTTP 201 status.
	gctx.JSON(http.StatusCreated, savedEvent)
}

func (h *handlers) GetEvents(gctx *gin.Context) {
	ctx := gctx.Request.Context()

	// Ready body
	body, err := io.ReadAll(gctx.Request.Body)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to read request body")
		gctx.AbortWithStatusJSON(http.StatusBadRequest, NewError("failed to read request body", err))

		return
	}

	// Checks that in GET requests there is no body
	if len(body) != 0 {
		log.Ctx(ctx).Error().Err(err).Msg("request body is not empty")
		gctx.AbortWithStatusJSON(http.StatusBadRequest, NewError("request body is not empty", err))

		return
	}

	// Checks that id param is there
	id := gctx.Param("id")
	if len(id) == 0 {
		log.Ctx(ctx).Error().Err(err).Msg("parameter 'id' is required")
		gctx.AbortWithStatusJSON(http.StatusBadRequest, NewError("parameter 'id' is required", err))

		return
	}

	// Get Event by ID: GET /events/{id}
	events, err := h.repository.GetEventById(ctx, id)
	if err != nil {
		// Checks that for empty slices, 404 is returned
		if errors.Is(err, ErrEventNotFound) {
			log.Ctx(ctx).Info().Msg("event not found")
			gctx.AbortWithStatusJSON(http.StatusNotFound, NewError("event not found", err))

			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("saving event failed")
		gctx.AbortWithStatusJSON(http.StatusBadRequest, NewError("saving event failed", err))

		return
	}

	// Returns the event with the specified UUID
	gctx.JSON(http.StatusOK, events)
}
