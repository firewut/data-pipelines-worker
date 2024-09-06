package functional_test

import (
	"data-pipelines-worker/api/handlers"
	"fmt"
	"net/http"
	"net/http/httptest"
)

func (suite *FunctionalTestSuite) TestPipelineStartHandler() {
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
