package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	"data-pipelines-worker/api/handlers"
	workerMiddleware "data-pipelines-worker/api/middleware"
	_ "data-pipelines-worker/docs"
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

type Server struct {
	sync.RWMutex

	host   string
	port   int
	config config.Config
	echo   *echo.Echo
	mdns   *types.MDNS

	Ready chan struct{}

	workerRegistry     interfaces.WorkerRegistry
	pipelineRegistry   interfaces.PipelineRegistry
	blockRegistry      interfaces.BlockRegistry
	processingRegistry interfaces.ProcessingRegistry
}

func NewServer(_config config.Config) *Server {
	workerRegistry := registries.GetWorkerRegistry(true)
	blockRegistry := registries.GetBlockRegistry(true)
	processingRegistry := registries.GetProcessingRegistry(true)

	pipelineRegistry, err := registries.NewPipelineRegistry(
		workerRegistry,
		blockRegistry,
		processingRegistry,
		dataclasses.NewPipelineCatalogueLoader(),
	)
	if err != nil {
		panic(err)
	}

	// Set the pipeline result storages
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			types.NewMINIOStorage(),
			types.NewLocalStorage(os.TempDir()),
		},
	)

	_echo := echo.New()
	_echo.HideBanner = true

	mdns := types.NewMDNS()

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
		Ready:              make(chan struct{}, 1),
	}
	worker.echo.Use(middleware.Logger())
	worker.echo.Use(middleware.Recover())
	worker.echo.Use(workerMiddleware.ConfigMiddleware(_config))

	return worker
}

func (s *Server) AddMiddleware(middleware ...echo.MiddlewareFunc) {
	s.Lock()
	defer s.Unlock()

	s.echo.Use(middleware...)
}

func (s *Server) AddHTTPAPIRoute(method string, path string, handlerFunc echo.HandlerFunc) {
	s.Lock()
	defer s.Unlock()

	s.echo.Add(method, path, handlerFunc)
}

func (s *Server) Start(ctx context.Context) {
	var cancel context.CancelFunc
	if ctx == nil {
		ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
	} else {
		cancel = func() {}
	}
	defer cancel()

	s.mdns.Announce()
	s.mdns.DiscoverWorkers()

	// Start server
	go func() {
		s.Ready <- struct{}{}

		if err := s.echo.Start(
			fmt.Sprintf("%s:%d", s.host, s.port),
		); err != nil && err != http.ErrServerClosed {
			s.echo.Logger.Fatal("shutting down the server due to an error:", err)
		}
	}()

	<-ctx.Done()

	s.Shutdown(time.Second * 5)
}

func (s *Server) Shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	defer close(s.Ready)

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
	s.Lock()
	defer s.Unlock()

	return s.echo.NewContext(request, writer)
}

func (s *Server) GetHost() string {
	s.Lock()
	defer s.Unlock()

	return s.host
}

func (s *Server) GetPort() int {
	s.Lock()
	defer s.Unlock()

	return s.port
}

func (s *Server) GetAPIAddress() string {
	return fmt.Sprintf(
		"http://%s",
		s.GetServerAddress().String(),
	)
}

func (s *Server) GetServerAddress() net.Addr {
	return s.GetEcho().ListenerAddr()
}

func (s *Server) SetPort(port int) {
	s.Lock()
	defer s.Unlock()

	s.port = port
}

func (s *Server) GetEcho() *echo.Echo {
	s.Lock()
	defer s.Unlock()

	return s.echo
}

func (s *Server) GetMDNS() *types.MDNS {
	s.Lock()
	defer s.Unlock()

	return s.mdns
}

func (s *Server) GetConfig() config.Config {
	s.Lock()
	defer s.Unlock()

	return s.config
}

func (s *Server) GetBlockRegistry() interfaces.BlockRegistry {
	s.Lock()
	defer s.Unlock()

	return s.blockRegistry
}

func (s *Server) GetPipelineRegistry() interfaces.PipelineRegistry {
	s.Lock()
	defer s.Unlock()

	return s.pipelineRegistry
}

func (s *Server) GetWorkerRegistry() interfaces.WorkerRegistry {
	s.Lock()
	defer s.Unlock()

	return s.workerRegistry
}

func (s *Server) GetProcessingRegistry() interfaces.ProcessingRegistry {
	s.Lock()
	defer s.Unlock()

	return s.processingRegistry
}

func (s *Server) SetAPIMiddlewares() {
	s.AddMiddleware(
		middleware.Logger(),
		middleware.Recover(),
		middleware.RequestID(),
		middleware.Gzip(),
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{
				"https://localhost",
				"https://localhost",
				"http://localhost:*",
				"http://127.0.0.1:*",
				"http://0.0.0.0.*",
			},
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		}),
	)
}

func (s *Server) SetAPIHandlers() {
	s.AddHTTPAPIRoute("GET", "/health", handlers.HealthHandler)
	s.AddHTTPAPIRoute("GET", "/blocks", handlers.BlocksHandler(
		s.GetBlockRegistry(),
	))
	s.AddHTTPAPIRoute("GET", "/workers", handlers.WorkersHandler(
		s.GetMDNS(),
	))
	s.AddHTTPAPIRoute("GET", "/pipelines", handlers.PipelinesHandler(
		s.GetPipelineRegistry(),
	))
	s.AddHTTPAPIRoute("GET", "/pipelines/:slug/", handlers.PipelineHandler(
		s.GetPipelineRegistry(),
	))
	s.AddHTTPAPIRoute("GET", "/pipelines/:slug/processings/:id", handlers.PipelineProcessingDetailsHandler(
		s.GetPipelineRegistry(),
	))
	s.AddHTTPAPIRoute("GET", "/pipelines/:slug/processings/", handlers.PipelineProcessingsStatusHandler(
		s.GetPipelineRegistry(),
	))
	s.AddHTTPAPIRoute(
		"POST", "/pipelines/:slug/start",
		handlers.PipelineStartHandler(
			s.GetPipelineRegistry(),
		),
	)
	s.AddHTTPAPIRoute(
		"POST", "/pipelines/:slug/resume",
		handlers.PipelineResumeHandler(
			s.GetPipelineRegistry(),
		),
	)

	if s.GetConfig().Swagger {
		s.AddHTTPAPIRoute(
			"GET", "/swagger/*",
			echoSwagger.WrapHandler,
		)
	}
}
