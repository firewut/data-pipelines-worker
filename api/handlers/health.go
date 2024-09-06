package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthHandler returns an HTTP handler function that responds with a simple
// "OK" message to indicate that the service is healthy.
//
// Parameters:
// - c: An echo.Context instance that provides context for the HTTP request.
//
// Returns:
// - An error if there is an issue writing the response, otherwise nil.
func HealthHandler(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
