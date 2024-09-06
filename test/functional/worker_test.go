package functional_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"data-pipelines-worker/api/handlers"
)

func (suite *FunctionalTestSuite) TestHealthHandler() {
	server := suite.GetServer()
	api_path := "/health"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)
	c.Set("Config", server.GetConfig())
	handlers.HealthHandler(c)

	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "OK")
}

func (suite *FunctionalTestSuite) TestBlocksHandler() {
	server := suite.GetServer()
	api_path := "/blocks"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)
	handlers.BlocksHandler(server.GetBlockRegistry())(c)

	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "http_request")
}

func (suite *FunctionalTestSuite) TestWorkersHandler() {
	server := suite.GetServer()
	api_path := "/workers"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)
	handlers.WorkersHandler(server.GetMDNS())(c)

	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "[]\n")
}

func (suite *FunctionalTestSuite) TestPipelinesHandler() {
	server := suite.GetServer()
	api_path := "/pipelines"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)
	handlers.PipelinesHandler(server.GetPipelineRegistry())(c)

	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "YT-CHANNEL-video-generation-block-prompt")
}
