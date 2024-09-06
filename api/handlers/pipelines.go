package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types/registries"
)

func PipelinesHandler(registry *registries.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, registry.GetAll())
	}
}
