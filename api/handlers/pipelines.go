package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/registries"
)

// PipelinesHandler returns an HTTP handler function that responds with a JSON
// representation of all pipelines in the provided PipelineRegistry.
//
// Parameters:
// - registry: A pointer to a PipelineRegistry instance containing the pipelines.
//
// Returns:
//   - An echo.HandlerFunc that handles HTTP requests and responds with a JSON
//     array of pipelines.
func PipelinesHandler(registry *registries.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, registry.GetAll())
	}
}

// PipelineStartHandler returns an HTTP handler function that responds with a JSON
// representation of the pipeline that was started.
//
// Parameters:
// - registry: A pointer to a PipelineRegistry instance containing the pipelines.
//
// Returns:
//   - An echo.HandlerFunc that handles HTTP requests and responds with a JSON
//     representation of the pipeline that was started.
func PipelineStartHandler(registry *registries.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		var inputData schemas.PipelineStartInputSchema
		if err := c.Bind(&inputData); err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		pipeline, err := registry.StartPipeline(inputData)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		return c.JSON(http.StatusOK, pipeline)
	}
}

// Handler to Resume processing of a pipeline which has `processing_id`
func PipelineResumeHandler(registry *registries.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		var inputData schemas.PipelineStartInputSchema
		if err := c.Bind(&inputData); err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		pipeline, err := registry.ResumePipeline(inputData)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		return c.JSON(http.StatusOK, pipeline)
	}
}
