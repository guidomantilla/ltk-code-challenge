package servers

import (
	"net"
	"net/http"

	"github.com/qmdx00/lifecycle"
)

var (
	_ Server = (*httpServer)(nil)
)

type Server interface {
	lifecycle.Server
}

type CronServer interface {
	Start()
	Stop()
}

type GrpcServer interface {
	Serve(lis net.Listener) error
	GracefulStop()
}

//

var (
	_ Application = (*lifecycle.App)(nil)
)

type Application interface {
	ID() string
	Name() string
	Version() string
	Metadata() map[string]string
	Attach(name string, server lifecycle.Server)
	Run() error
}

//

var (
	_ BuildHttpServerFn = BuildHttpServer
)

type BuildHttpServerFn func(server *http.Server) (string, Server)
