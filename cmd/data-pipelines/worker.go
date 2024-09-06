package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"data-pipelines-worker/api"
	"data-pipelines-worker/api/handlers"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	worker := api.NewServer()
	defer worker.Shutdown(time.Second * 5)

	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal: %s. Shutting down...\n", sig)
		// Perform cleanup or shutdown tasks here
		worker.Shutdown(time.Second * 5) // Assuming you have a Stop method to clean up
		os.Exit(0)
	}()

	worker.AddMiddleware(
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

	worker.AddHTTPAPIRoute("GET", "/health", handlers.HealthHandler)
	worker.AddHTTPAPIRoute("GET", "/blocks", handlers.BlocksHandler(
		worker.GetBlockRegistry(),
	))
	worker.AddHTTPAPIRoute("GET", "/workers", handlers.WorkersHandler(
		worker.GetMDNS(),
	))
	worker.AddHTTPAPIRoute("GET", "/pipelines", handlers.PipelinesHandler(
		worker.GetPipelineRegistry(),
	))
	worker.AddHTTPAPIRoute(
		"POST", "/pipelines/:slug/start",
		handlers.PipelineStartHandler(
			worker.GetPipelineRegistry(),
		),
	)

	worker.Start()
}
