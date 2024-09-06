package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types"
)

// WorkersHandler returns an HTTP handler function that responds with a JSON
// representation of all discovered workers in the provided MDNS instance.
//
// Parameters:
// - mDNS: A pointer to an MDNS instance containing the discovered workers.
//
// Returns:
//   - An echo.HandlerFunc that handles HTTP requests and responds with a JSON
//     array of discovered workers.
func WorkersHandler(mDNS *types.MDNS) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, mDNS.GetDiscoveredWorkers())
	}
}
