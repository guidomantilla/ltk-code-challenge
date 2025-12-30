package servers

import (
	"context"
	"errors"
	"net/http"

	"github.com/qmdx00/lifecycle"
	"github.com/rs/zerolog/log"
)

type httpServer struct {
	ctx      context.Context //nolint:containedctx
	name     string
	internal *http.Server
}

func BuildHttpServer(server *http.Server) (string, Server) {
	return "http-server", NewHttpServer(server)
}

func NewHttpServer(server *http.Server) lifecycle.Server {
	return &httpServer{
		name:     "http-server",
		internal: server,
	}
}

func (server *httpServer) Run(ctx context.Context) error {
	log.Ctx(ctx).Info().Str("stage", "startup").Str("component", server.name).Msg("starting up")

	server.ctx = ctx

	err := server.internal.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Ctx(ctx).Error().Str("stage", "startup").Str("component", server.name).Err(err).Msg("failed to listen or serve")
		return ErrServerFailedToStart(server.name, err)
	}

	return nil
}

func (server *httpServer) Stop(ctx context.Context) error {
	log.Ctx(ctx).Info().Str("stage", "shut down").Str("component", server.name).Msg("stopping")
	defer log.Ctx(ctx).Info().Str("stage", "shut down").Str("component", server.name).Msg("stopped")

	err := server.internal.Shutdown(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Str("stage", "shut down").Str("component", server.name).Err(err).Msg("failed to stop")
		return ErrServerFailedToStop(server.name, err)
	}

	return nil
}
