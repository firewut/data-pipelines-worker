package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/interfaces"
)

// @Summary Get all pipelines
// @Description Returns a JSON array of all pipelines in the registry.
// @Tags pipelines
// @Accept json
// @Produce json
// @Success 200 {array} dataclasses.PipelineData
// @Router /pipelines [get]
func PipelinesHandler(registry interfaces.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, registry.GetAll())
	}
}

// @Summary Get a pipeline
// @Description Returns a JSON object of the pipeline with the given slug.
// @Tags pipelines
// @Accept json
// @Produce json
// @Param slug path string true "Pipeline slug"
// @Success 200 {object} dataclasses.PipelineData
// @Failure 404 {string} string "Pipeline not found"
// @Router /pipelines/{slug} [get]
func PipelineHandler(registry interfaces.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		pipeline := registry.Get(c.Param("slug"))
		if pipeline == nil {
			return c.JSON(http.StatusNotFound, "Pipeline not found")
		}

		return c.JSON(http.StatusOK, pipeline)
	}
}

// @Summary Get pipeline Processings info
// @Description Returns a JSON object of the pipeline Processings.
// @Tags pipelines
// @Accept json
// @Produce json
// @Param slug path string true "Pipeline slug"
// @Success 200 {object} map[uuid.UUID][]dataclasses.PipelineProcessingStatus
// @Failure 404 {string} string "Pipeline not found"
// @Router /pipelines/{slug}/processings/ [get]
func PipelineProcessingsStatusHandler(registry interfaces.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		pipeline := registry.Get(c.Param("slug"))
		if pipeline == nil {
			return c.JSON(http.StatusNotFound, "Pipeline not found")
		}

		return c.JSON(http.StatusOK, registry.GetProcessingsStatus(pipeline))
	}
}

// @Summary Get pipeline Processing Details by Log Id
// @Description Returns a JSON object of the pipeline Processing Details.
// @Tags pipelines
// @Accept json
// @Produce json
// @Param slug path string true "Pipeline slug"
// @Param id path string true "Processing ID"
// @Param log-id path string true "Log ID"
// @Success 200 {object} dataclasses.PipelineProcessingDetails
// @Failure 404 {string} string "Pipeline not found"
// @Router /pipelines/{slug}/processings/{id}/{log-id} [get]
func PipelineProcessingDetailsByLogIdHandler(registry interfaces.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		pipeline := registry.Get(c.Param("slug"))
		id := c.Param("id")
		if pipeline == nil {
			return c.JSON(http.StatusNotFound, "Pipeline not found")
		}
		if len(id) == 0 {
			return c.JSON(http.StatusNotFound, "Processing not found")
		}
		processingId, err := uuid.Parse(id)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "Invalid processing ID")
		}
		logId, err := uuid.Parse(c.Param("log-id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, "Invalid log ID")
		}

		return c.JSON(http.StatusOK, registry.GetProcessingDetailsByLogId(pipeline, processingId, logId))
	}
}

// @Summary Get pipeline Processing Details
// @Description Returns a JSON object of the pipeline Processing Details.
// @Tags pipelines
// @Accept json
// @Produce json
// @Param slug path string true "Pipeline slug"
// @Param id path string true "Processing ID"
// @Success 200 {object} []dataclasses.PipelineProcessingDetails
// @Failure 404 {string} string "Pipeline not found"
// @Router /pipelines/{slug}/processings/{id} [get]
func PipelineProcessingDetailsHandler(registry interfaces.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		pipeline := registry.Get(c.Param("slug"))
		id := c.Param("id")
		if pipeline == nil {
			return c.JSON(http.StatusNotFound, "Pipeline not found")
		}
		if len(id) == 0 {
			return c.JSON(http.StatusNotFound, "Processing not found")
		}
		processingId, err := uuid.Parse(id)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "Invalid processing ID")
		}

		return c.JSON(http.StatusOK, registry.GetProcessingDetails(pipeline, processingId))
	}
}

// @Summary Start a pipeline
// @Description Starts the pipeline with the given input data and returns the processing ID.
// @Tags pipelines
// @Accept json, multipart/form-data
// @Produce json
// @Param slug path string true "Pipeline slug"
// @Param input body schemas.PipelineStartInputSchema true "Input data to start the pipeline"
// @Success 200 {object} schemas.PipelineStartOutputSchema
// @Failure 400 {string} string "Bad request"
// @Router /pipelines/{slug}/start [post]
func PipelineStartHandler(registry interfaces.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		var inputData schemas.PipelineStartInputSchema

		// Check the Content-Type to determine how to bind the request
		contentType := c.Request().Header.Get("Content-Type")
		switch {
		case contentType == "application/json":
			// Bind JSON payload directly
			if err := c.Bind(&inputData); err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
		case strings.HasPrefix(contentType, "multipart/form-data"):
			// For multipart/form-data, parse the form data
			if err := c.Request().ParseMultipartForm(10 << 20); err != nil {
				return c.JSON(http.StatusBadRequest, "Unable to parse multipart form")
			}

			// Parse the form data and files into the `inputData` struct
			if err := inputData.ParseForm(c.Request()); err != nil {
				return c.JSON(http.StatusBadRequest, fmt.Sprintf("Error parsing pipeline data: %v", err))
			}
		default:
			// Unsupported content type
			return c.JSON(http.StatusBadRequest, "Unsupported Content-Type")
		}

		// Update request pipeline slug from url
		inputData.Pipeline.Slug = c.Param("slug")

		processingId, err := registry.StartPipeline(inputData)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		return c.JSON(
			http.StatusOK,
			schemas.PipelineStartOutputSchema{
				ProcessingID: processingId,
			},
		)
	}
}

// @Summary Resume a paused pipeline
// @Description Resumes a paused pipeline with the given input data and returns the processing ID.
// @Tags pipelines
// @Accept json, multipart/form-data
// @Produce json
// @Param slug path string true "Pipeline slug"
// @Param input body schemas.PipelineStartInputSchema true "Input data to resume the pipeline"
// @Success 200 {object} schemas.PipelineResumeOutputSchema
// @Failure 400 {string} string "Bad request"
// @Router /pipelines/{slug}/resume [post]
func PipelineResumeHandler(registry interfaces.PipelineRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		var inputData schemas.PipelineStartInputSchema

		// Check the Content-Type to determine how to bind the request
		contentType := c.Request().Header.Get("Content-Type")
		switch {
		case contentType == "application/json":
			// Bind JSON payload directly
			if err := c.Bind(&inputData); err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
		case strings.HasPrefix(contentType, "multipart/form-data"):
			// For multipart/form-data, parse the form data
			if err := c.Request().ParseMultipartForm(10 << 20); err != nil {
				return c.JSON(http.StatusBadRequest, "Unable to parse multipart form")
			}

			if err := inputData.ParseForm(c.Request()); err != nil {
				return c.JSON(http.StatusBadRequest, fmt.Sprintf("Error parsing pipeline data: %v", err))
			}
		default:
			// Unsupported content type
			return c.JSON(http.StatusBadRequest, "Unsupported Content-Type")
		}

		// Update request pipeline slug from url
		inputData.Pipeline.Slug = c.Param("slug")

		processingId, err := registry.ResumePipeline(inputData)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		return c.JSON(
			http.StatusOK,
			schemas.PipelineResumeOutputSchema{
				ProcessingID: processingId,
			},
		)
	}
}
