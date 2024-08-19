package api

import (
	"context"
	workerMiddleware "data-pipelines-worker/api/middleware"
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/blocks"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	host   string
	port   int
	config types.Config

	echo *echo.Echo
	mdns *types.MDNS
}

func NewServer() *Server {
	config := types.GetConfig()
	detectedBlocks := blocks.DetectBlocks()

	_echo := echo.New()
	_echo.HideBanner = true

	mdns := types.NewMDNS(config)
	mdns.SetDetectedBlocks(detectedBlocks)

	var worker = &Server{
		host:   config.HTTPAPIServer.Host,
		port:   config.HTTPAPIServer.Port,
		echo:   _echo,
		mdns:   mdns,
		config: config,
	}
	worker.echo.Use(middleware.Logger())
	worker.echo.Use(middleware.Recover())
	worker.echo.Use(
		workerMiddleware.ConfigMiddleware(config),
	)

	return worker
}

func (s *Server) AddMiddleware(middleware ...echo.MiddlewareFunc) {
	s.echo.Use(middleware...)
}

func (s *Server) AddHTTPAPIRoute(method string, path string, handlerFunc echo.HandlerFunc) {
	s.echo.Add(method, path, handlerFunc)
}

func (s *Server) Start() {
	s.mdns.Announce()
	s.mdns.DiscoverWorkers()
	s.echo.Logger.Fatal(
		s.echo.Start(fmt.Sprintf("%s:%d", s.host, s.port)),
	)
}

func (s *Server) Shutdown(timeout time.Duration) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	s.mdns.Shutdown()
	s.echo.Shutdown(ctx)
}

func (s *Server) NewContext(request *http.Request, writer http.ResponseWriter) echo.Context {
	return s.echo.NewContext(request, writer)
}

func (s *Server) GetHost() string {
	return s.host
}

func (s *Server) GetPort() int {
	return s.port
}

func (s *Server) GetEcho() *echo.Echo {
	return s.echo
}

func (s *Server) GetMDNS() *types.MDNS {
	return s.mdns
}

func (s *Server) GetConfig() types.Config {
	return s.config
}

func (s *Server) GetDetectedBlocks() []types.Block {
	return s.mdns.GetDetectedBlocks()
}
