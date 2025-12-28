package servers

import (
	"context"

	"github.com/qmdx00/lifecycle"
	"github.com/rs/zerolog/log"

	"ltk-code-challenge/pkg/resources"
)

type baseServer struct {
	ctx          context.Context //nolint:containedctx
	name         string
	closeChannel chan struct{}
	closables    []resources.Closable
}

func BuildBaseServer(closables ...resources.Closable) (string, Server) {
	return "base-server", NewBaseServer(closables...)
}

func NewBaseServer(closables ...resources.Closable) lifecycle.Server {
	return &baseServer{
		name:         "base-server",
		closeChannel: make(chan struct{}),
		closables:    closables,
	}
}

func (server *baseServer) Run(ctx context.Context) error {
	log.Info().Str("stage", "startup").Str("component", server.name).Msg("starting up")

	server.ctx = ctx
	<-server.closeChannel

	return nil
}

func (server *baseServer) Stop(_ context.Context) error {
	log.Info().Str("stage", "shut down").Str("component", server.name).Msg("stopping")
	defer log.Info().Str("stage", "shut down").Str("component", server.name).Msg("stopped")

	for _, closable := range server.closables {
		closable.Close()
	}

	close(server.closeChannel)

	return nil
}
