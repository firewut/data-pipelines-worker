package main

import (
	"time"

	"data-pipelines-worker/api"
	"data-pipelines-worker/api/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	worker := api.NewWorker()
	defer worker.Shutdown(time.Second * 5)

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

	worker.AddHTTPAPIRoute("GET", "/workers/discover", handlers.WorkersDiscoverHandler)

	worker.Start()
}
