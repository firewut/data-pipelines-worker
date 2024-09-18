package functional_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/api/handlers"
	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/registries"
)

func (suite *FunctionalTestSuite) TestPipelineStartHandler() {
	// Given
	server := suite.GetServer()
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
	suite.Contains(rec.Body.String(), "test-two-http-blocks")
}

func (suite *FunctionalTestSuite) TestPipelineStartHandlerTwoBlocks() {
	// Given
	server := suite.GetServer()
	api_path := "/pipelines"

	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, time.Millisecond)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	payload, err := json.Marshal(
		&schemas.PipelineStartInputSchema{
			Pipeline: schemas.PipelineInputSchema{
				Slug: "test-two-http-blocks",
			},
			Block: schemas.BlockInputSchema{
				Slug: "test-block-first-slug",
				Input: map[string]interface{}{
					"url": firstBlockInput,
				},
			},
		},
	)
	suite.Nil(err)
	suite.NotEmpty(payload)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/%s", api_path),
		bytes.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")

	c := server.GetEcho().NewContext(req, rec)
	c.SetParamNames("slug")
	c.SetParamValues("test-two-http-blocks")

	var response schemas.PipelineStartOutputSchema

	// When
	handlers.PipelineStartHandler(server.GetPipelineRegistry())(c)

	// Then
	suite.Equal(http.StatusOK, rec.Code, rec.Body.String())
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	suite.Nil(err)
	suite.NotEmpty(response.ProcessingID)

	resultStorages := registries.NewPipelineBlockDataRegistry(
		response.ProcessingID,
		"test-two-http-blocks",
		server.GetPipelineRegistry().GetPipelineResultStorages(),
	)
	firstBlockOutput := resultStorages.LoadOutput("test-block-first-slug")
	suite.NotEmpty(firstBlockOutput)
	suite.Equal(secondBlockInput, firstBlockOutput[0].String())
	secondBlockOutput := resultStorages.LoadOutput("test-block-second-slug")
	suite.NotEmpty(secondBlockOutput)
	suite.Equal(mockedSecondBlockResponse, secondBlockOutput[0].String())
}
