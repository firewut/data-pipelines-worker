package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types"
)

func BlocksHandler(detectedBlocks []types.Block) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, detectedBlocks)
	}
}
