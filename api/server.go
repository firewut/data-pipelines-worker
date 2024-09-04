package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	workerMiddleware "data-pipelines-worker/api/middleware"
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

type Server struct {
	host   string
	port   int
	config config.Config

	echo             *echo.Echo
	mdns             *types.MDNS
	pipelineRegistry *registries.PipelineRegistry
	blockRegistry    *registries.BlockRegistry
}

func NewServer() *Server {
	_config := config.GetConfig()

	pipelineRegistry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	if err != nil {
		panic(err)
	}

	blockRegistry := registries.GetBlockRegistry()

	_echo := echo.New()
	_echo.HideBanner = true

	mdns := types.NewMDNS(_config)
	mdns.SetDetectedBlocks(blockRegistry.GetBlocks())

	var worker = &Server{
		host:             _config.HTTPAPIServer.Host,
		port:             _config.HTTPAPIServer.Port,
		echo:             _echo,
		mdns:             mdns,
		config:           _config,
		pipelineRegistry: pipelineRegistry,
		blockRegistry:    blockRegistry,
	}
	worker.echo.Use(middleware.Logger())
	worker.echo.Use(middleware.Recover())
	worker.echo.Use(
		workerMiddleware.ConfigMiddleware(_config),
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
	s.pipelineRegistry.Shutdown()
	s.blockRegistry.Shutdown()

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

func (s *Server) GetConfig() config.Config {
	return s.config
}

func (s *Server) GetDetectedBlocks() map[string]interfaces.Block {
	return s.mdns.GetDetectedBlocks()
}
