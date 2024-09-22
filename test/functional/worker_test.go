package functional_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"data-pipelines-worker/api/handlers"
	"data-pipelines-worker/test/factories"
)

func (suite *FunctionalTestSuite) TestHealthHandler() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	server, _, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)

	api_path := "/health"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)
	c.Set("Config", server.GetConfig())

	// When
	handlers.HealthHandler(c)

	// Then
	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "OK")
}

func (suite *FunctionalTestSuite) TestBlocksHandler() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	server, _, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)
	api_path := "/blocks"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)

	// When
	handlers.BlocksHandler(server.GetBlockRegistry())(c)

	// Then
	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "http_request")
}

func (suite *FunctionalTestSuite) TestWorkersHandler() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	server, _, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)
	api_path := "/workers"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)

	// When
	handlers.WorkersHandler(server.GetMDNS())(c)

	// Then
	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "[]\n")
}

func (suite *FunctionalTestSuite) TestPipelinesHandler() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	server, _, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)
	api_path := "/pipelines"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)

	// When
	handlers.PipelinesHandler(server.GetPipelineRegistry())(c)

	// Then
	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "YT-CHANNEL-video-generation-block-prompt")
}
