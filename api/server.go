package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

	echo               *echo.Echo
	mdns               *types.MDNS
	workerRegistry     interfaces.Registry[interfaces.Worker]
	pipelineRegistry   interfaces.Registry[interfaces.Pipeline]
	blockRegistry      interfaces.Registry[interfaces.Block]
	processingRegistry interfaces.Registry[interfaces.Processing]
}

func NewServer() *Server {
	_config := config.GetConfig()

	workerRegistry := registries.GetWorkerRegistry()

	pipelineRegistry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	// Set the pipeline result storages
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			types.NewLocalStorage(os.TempDir()),
			types.NewMINIOStorage(),
		},
	)
	if err != nil {
		panic(err)
	}

	blockRegistry := registries.GetBlockRegistry()
	processingRegistry := registries.GetProcessingRegistry()

	_echo := echo.New()
	_echo.HideBanner = true

	mdns := types.NewMDNS()
	mdns.SetBlocks(blockRegistry.GetAll())

	var worker = &Server{
		host:               _config.HTTPAPIServer.Host,
		port:               _config.HTTPAPIServer.Port,
		echo:               _echo,
		mdns:               mdns,
		config:             _config,
		workerRegistry:     workerRegistry,
		pipelineRegistry:   pipelineRegistry,
		blockRegistry:      blockRegistry,
		processingRegistry: processingRegistry,
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	shutdownCalls := []func(context.Context) error{
		s.mdns.Shutdown,
		s.blockRegistry.Shutdown,
		s.pipelineRegistry.Shutdown,
	}

	for _, shutdownCall := range shutdownCalls {
		callCtx, cancelCtx := context.WithTimeout(context.Background(), timeout)
		defer cancelCtx()

		shutdownCall(callCtx)
	}

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

func (s *Server) GetBlockRegistry() *registries.BlockRegistry {
	return s.blockRegistry.(*registries.BlockRegistry)
}

func (s *Server) GetPipelineRegistry() *registries.PipelineRegistry {
	return s.pipelineRegistry.(*registries.PipelineRegistry)
}

func (s *Server) GetWorkerRegistry() *registries.WorkerRegistry {
	return s.workerRegistry.(*registries.WorkerRegistry)
}

func (s *Server) GetProcessingRegistry() *registries.ProcessingRegistry {
	return s.processingRegistry.(*registries.ProcessingRegistry)
}
