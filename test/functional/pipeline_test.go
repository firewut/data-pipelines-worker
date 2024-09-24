package functional_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/api/handlers"
	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/registries"
)

func (suite *FunctionalTestSuite) TestPipelineStartHandler() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	server, _, err := factories.NewWorkerServerWithHandlers(ctx, true)
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
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	testPipelineSlug, _ := "test-two-http-blocks", "http_request"

	server, _, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)
	suite.NotEmpty(server)
	server.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))

	serverProcessingRegistry := server.GetProcessingRegistry()

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, time.Millisecond)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

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

	// Wait for two blocks to process
	block1Processing := <-serverProcessingRegistry.GetProcessingCompletedChannel()
	suite.NotEmpty(block1Processing.GetId())
	suite.Equal(processingResponse.ProcessingID, block1Processing.GetId())
	block2Processing := <-serverProcessingRegistry.GetProcessingCompletedChannel()
	suite.NotEmpty(block1Processing.GetId())
	suite.Equal(processingResponse.ProcessingID, block2Processing.GetId())

	// Check server storages
	resultStorages := registries.NewPipelineBlockDataRegistry(
		processingResponse.ProcessingID,
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
