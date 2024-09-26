package functional_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"

	"data-pipelines-worker/api/handlers"
	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/interfaces"
)

func (suite *FunctionalTestSuite) TestPipelineStartHandler() {
	// Given
	server, _, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)

	api_path := "/pipelines"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)
	server.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))

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
	server, _, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	suite.NotEmpty(server)

	notificationChannel := make(chan interfaces.Processing)
	serverProcessingRegistry := server.GetProcessingRegistry()
	serverProcessingRegistry.SetNotificationChannel(notificationChannel)

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)
	server.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(firstBlockInput))

	testPipelineSlug, _ := "test-two-http-blocks", "http_request"
	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: testPipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: "test-block-first-slug",
			Input: map[string]interface{}{
				"url": firstBlockInput,
			},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	// Wait for completion events
	block1Processing := <-notificationChannel
	suite.NotEmpty(block1Processing.GetId())
	suite.Equal(processingResponse.ProcessingID, block1Processing.GetId())

	block2Processing := <-notificationChannel
	suite.NotEmpty(block2Processing.GetId())
	suite.Equal(processingResponse.ProcessingID, block2Processing.GetId())
}
