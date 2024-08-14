package handlers

import (
	"data-pipelines-worker/types"
	"net/http"

	"github.com/labstack/echo/v4"
)

func BlocksHandler(detectedBlocks []types.Block) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, detectedBlocks)
	}
}
