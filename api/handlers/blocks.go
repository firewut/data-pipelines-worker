package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/types/registries"
)

// BlocksHandler returns an HTTP handler function that responds with a JSON
// representation of all blocks in the provided BlockRegistry.
//
// Parameters:
// - registry: A pointer to a BlockRegistry instance containing the blocks.
//
// Returns:
//   - An echo.HandlerFunc that handles HTTP requests and responds with a JSON
//     array of blocks.
func BlocksHandler(registry *registries.BlockRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, registry.GetBlocks())
	}
}
