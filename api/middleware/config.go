package middlewares

import (
	"data-pipelines-worker/types"

	"github.com/labstack/echo/v4"
)

func ConfigMiddleware(config types.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("Config", config)

			return next(c)
		}
	}
}
