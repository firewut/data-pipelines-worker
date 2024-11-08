package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// @Summary Check service health
// @Description Responds with a simple "OK" message to indicate that the service is healthy.
// @Tags health
// @Accept plain
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /health [get]
func HealthHandler(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
