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

type Worker struct {
	host   string
	port   int
	config types.Config

	echo *echo.Echo
	mdns *types.MDNS
}

func NewWorker() *Worker {
	detectedBlocksIds := make([]string, 0)

	config := types.NewConfig()
	detectedBlocks := blocks.DetectBlocks()

	_echo := echo.New()
	_echo.HideBanner = true

	mdns := types.NewMDNS(config)
	for _, block := range detectedBlocks {
		detectedBlocksIds = append(detectedBlocksIds, block.GetId())
	}
	mdns.SetDetectedBlocks(detectedBlocksIds)

	var worker = &Worker{
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

func (w *Worker) AddMiddleware(middleware ...echo.MiddlewareFunc) {
	w.echo.Use(middleware...)
}

func (w *Worker) AddHTTPAPIRoute(method string, path string, handlerFunc echo.HandlerFunc) {
	w.echo.Add(method, path, handlerFunc)
}

func (w *Worker) Start() {
	w.mdns.Advertise()
	w.echo.Logger.Fatal(
		w.echo.Start(fmt.Sprintf("%s:%d", w.host, w.port)),
	)
}

func (w *Worker) Shutdown(timeout time.Duration) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	w.mdns.Shutdown()
	w.echo.Shutdown(ctx)
}

func (w *Worker) NewContext(request *http.Request, writer http.ResponseWriter) echo.Context {
	return w.echo.NewContext(request, writer)
}

func (w *Worker) GetHost() string {
	return w.host
}

func (w *Worker) GetPort() int {
	return w.port
}

func (w *Worker) GetEcho() *echo.Echo {
	return w.echo
}

func (w *Worker) GetConfig() types.Config {
	return w.config
}
