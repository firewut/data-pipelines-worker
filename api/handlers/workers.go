package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types"
)

// @Summary Get all discovered workers
// @Description Returns a JSON array of all discovered workers in the mDNS instance.
// @Tags workers
// @Accept json
// @Produce json
// @Success 200 {array} dataclasses.Worker
// @Router /workers [get]
func WorkersHandler(mDNS *types.MDNS) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, mDNS.GetDiscoveredWorkers())
	}
}
