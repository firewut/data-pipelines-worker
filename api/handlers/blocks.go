package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types/interfaces"
)

// @Summary Get all blocks
// @Description Returns a JSON array of all blocks in the registry.
// @Tags blocks
// @Accept json
// @Produce json
// @Success 200 {array} interfaces.Block
// @Router /blocks [get]
func BlocksHandler(registry interfaces.BlockRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, registry.GetAll())
	}
}
