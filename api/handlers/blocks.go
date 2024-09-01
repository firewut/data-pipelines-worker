package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types/interfaces"
)

func BlocksHandler(detectedBlocks map[string]interfaces.Block) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, detectedBlocks)
	}
}
