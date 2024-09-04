package middlewares

import (
	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types/config"
)

func ConfigMiddleware(config config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("Config", config)

			return next(c)
		}
	}
}
