package functional_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/interfaces"
)

func (suite *FunctionalTestSuite) TestWorkerShutdownCorrect() {
	// Given
	httpClient := &http.Client{}
	server1, _, err := suite.NewWorkerServerWithHandlers(true)

	// When
	suite.Nil(err)
	response, err := httpClient.Get(
		fmt.Sprintf(
			"%s/health",
			server1.GetAPIAddress(),
		),
	)

	// Then
	suite.Nil(err)
	suite.Equal(http.StatusOK, response.StatusCode)
}

func (suite *FunctionalTestSuite) TestTwoWorkersDifferentRegistries() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	// When
	server1, worker1, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)
	server2, worker2, err := factories.NewWorkerServerWithHandlers(ctx, false)
	suite.Nil(err)

	// Then
	suite.False(server1.GetWorkerRegistry() == server2.GetWorkerRegistry())
	suite.False(server1.GetPipelineRegistry() == server2.GetPipelineRegistry())
	suite.False(server1.GetBlockRegistry() == server2.GetBlockRegistry())
	suite.False(server1.GetProcessingRegistry() == server2.GetProcessingRegistry())

	server1.GetWorkerRegistry().Add(worker1)
	server1.GetWorkerRegistry().Add(worker2)

	suite.NotEqual(
		server1.GetWorkerRegistry().GetAll(),
		server2.GetWorkerRegistry().GetAll(),
	)
}

func (suite *FunctionalTestSuite) TestTwoWorkersAPIDiscoveryCommunication() {
	// Given
	testPipelineSlug, testBlockId := "test-two-http-blocks", "http_request"
	server1, worker1, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server1.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))

	server2, worker2, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server2.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))

	workerRegistry1 := server1.GetWorkerRegistry()
	workerRegistry2 := server2.GetWorkerRegistry()

	// When
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	// Then
	availableWorkers1 := workerRegistry1.GetAvailableWorkers()
	availableWorkers2 := workerRegistry2.GetAvailableWorkers()
	suite.Equal(len(availableWorkers1), 1)
	suite.Equal(len(availableWorkers2), 1)
	suite.Equal(availableWorkers1[worker2.GetId()], worker2)
	suite.Equal(availableWorkers2[worker1.GetId()], worker1)

	workersWithBlocks1 := workerRegistry1.GetWorkersWithBlocksDetected(availableWorkers1, testBlockId)
	workersWithBlocks2 := workerRegistry2.GetWorkersWithBlocksDetected(availableWorkers2, testBlockId)
	suite.Equal(len(workersWithBlocks1), 1)
	suite.Equal(len(workersWithBlocks2), 1)
	suite.Equal(workersWithBlocks1[worker2.GetId()], worker2)
	suite.Equal(workersWithBlocks2[worker1.GetId()], worker1)

	validWorkers1 := workerRegistry1.GetValidWorkers(testPipelineSlug, testBlockId)
	validWorkers2 := workerRegistry2.GetValidWorkers(testPipelineSlug, testBlockId)
	suite.Equal(len(validWorkers1), 1)
	suite.Equal(len(validWorkers2), 1)
	suite.Equal(validWorkers1[worker2.GetId()], worker2)
	suite.Equal(validWorkers2[worker1.GetId()], worker1)
}

func (suite *FunctionalTestSuite) TestTwoWorkersPipelineProcessingRequiredBlocksDisabledEverywhere() {
	// Given
	testPipelineSlug, testBlockId := "test-two-http-blocks", "http_request"
	server1, worker1, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server1.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))
	server2, worker2, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server2.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))

	notificationChannel := make(chan interfaces.Processing)
	processingRegistry1 := server1.GetProcessingRegistry()
	processingRegistry1.SetNotificationChannel(notificationChannel)

	workerRegistry1 := server1.GetWorkerRegistry()
	workerRegistry2 := server2.GetWorkerRegistry()
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	blockRegistry1 := server1.GetBlockRegistry()
	blockRegistry2 := server2.GetBlockRegistry()
	blockRegistry1.GetAvailableBlocks()[testBlockId].SetAvailable(false)
	blockRegistry2.GetAvailableBlocks()[testBlockId].SetAvailable(false)

	suite.NotContains(blockRegistry1.GetAvailableBlocks(), testBlockId) // Ensure block is deleted
	suite.NotContains(blockRegistry2.GetAvailableBlocks(), testBlockId)

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
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
		server1,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	failedProcessing1 := <-notificationChannel
	suite.Equal(failedProcessing1.GetId(), processingResponse.ProcessingID)

	processing1 := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processing1)

	suite.NotNil(processing1.GetError())
	suite.Equal(interfaces.ProcessingStatusFailed, processing1.GetStatus())
}

func (suite *FunctionalTestSuite) TestTwoWorkersPipelineProcessingRequiredBlocksDisabledFirst() {
	// Given
	testPipelineSlug, testBlockId := "test-two-http-blocks", "http_request"
	server1, worker1, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server1.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))
	server2, worker2, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server2.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))

	workerRegistry1 := server1.GetWorkerRegistry()
	workerRegistry2 := server2.GetWorkerRegistry()
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	notificationChannel1 := make(chan interfaces.Processing)
	notificationChannel2 := make(chan interfaces.Processing)

	processingRegistry1 := server1.GetProcessingRegistry()
	processingRegistry2 := server2.GetProcessingRegistry()

	processingRegistry1.SetNotificationChannel(notificationChannel1)
	processingRegistry2.SetNotificationChannel(notificationChannel2)

	blockRegistry1 := server1.GetBlockRegistry()
	blockRegistry2 := server2.GetBlockRegistry()
	blockRegistry1.GetAvailableBlocks()[testBlockId].SetAvailable(false)
	blockRegistry2.GetAvailableBlocks()[testBlockId].SetAvailable(true)

	suite.NotContains(blockRegistry1.GetAvailableBlocks(), testBlockId) // Ensure block is deleted
	suite.Contains(blockRegistry2.GetAvailableBlocks(), testBlockId)

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
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
		server1,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	transferredProcessing1 := <-notificationChannel1
	suite.Equal(transferredProcessing1.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusTransferred, transferredProcessing1.GetStatus())
	suite.Nil(transferredProcessing1.GetError())

	processingRegistry1Processing := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry1Processing)
	suite.Equal(transferredProcessing1.GetId(), processingRegistry1Processing.GetId())

	completedProcessing21 := <-notificationChannel2
	suite.Equal(completedProcessing21.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing21.GetStatus())

	processingRegistry2Processing1 := processingRegistry2.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry2Processing1)
	suite.Equal(completedProcessing21.GetId(), processingRegistry2Processing1.GetId())
	suite.Equal(secondBlockInput, completedProcessing21.GetOutput().GetValue().String())

	completedProcessing22 := <-notificationChannel2
	suite.Equal(completedProcessing22.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing22.GetStatus())

	processingRegistry2Processing2 := processingRegistry2.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry2Processing2)
	suite.Equal(completedProcessing22.GetId(), processingRegistry2Processing2.GetId())
	suite.Equal(mockedSecondBlockResponse, completedProcessing22.GetOutput().GetValue().String())
}

func (suite *FunctionalTestSuite) TestTwoWorkersPipelineProcessingWorker1HasNoSecondBlock() {
	// Given
	imageWidth, imageHeight := 100, 100
	testPipelineSlug := "test-two-http-blocks"
	firstWorkerDisabledBlocks := []string{"image_add_text"}
	secondWorkerDisabledBlocks := []string{"http_request"}
	imageContent := suite.GetPNGImageBuffer(imageWidth, imageHeight)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(imageContent.Bytes())
	}))
	suite.httpTestServers = append(suite.httpTestServers, server)
	imageUrl := server.URL

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(`{
			"slug": "%s",
			"title": "Test two Blocks",
			"description": "First Block downloads image and second block adds text to it",
			"blocks": [
				{
					"id": "http_request",
					"slug": "test-block-first-slug",
					"description": "Download Image from provided URL",
					"input": {
						"url": "%s"
					}
				},
				{
					"id": "image_add_text",
					"slug": "test-block-second-slug",
					"description": "Add text to downloaded image",
					"input_config": {
						"property": {
							"image": {
								"origin": "test-block-first-slug"
							}
						}
					},
					"input": {
						"text": "Hello, world!",
						"font_size": 50,
						"font_color": "#000000"
					}
				}
			]
		}`,
			testPipelineSlug,
			imageUrl,
		),
	)

	server1, worker1, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server1.GetPipelineRegistry().Add(pipeline)
	server2, worker2, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server2.GetPipelineRegistry().Add(pipeline)

	workerRegistry1 := server1.GetWorkerRegistry()
	workerRegistry2 := server2.GetWorkerRegistry()
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	notificationChannel1 := make(chan interfaces.Processing)
	notificationChannel2 := make(chan interfaces.Processing)

	processingRegistry1 := server1.GetProcessingRegistry()
	processingRegistry2 := server2.GetProcessingRegistry()

	processingRegistry1.SetNotificationChannel(notificationChannel1)
	processingRegistry2.SetNotificationChannel(notificationChannel2)

	blockRegistry1 := server1.GetBlockRegistry()
	blockRegistry2 := server2.GetBlockRegistry()

	for _, blockId := range firstWorkerDisabledBlocks {
		blockRegistry1.GetAvailableBlocks()[blockId].SetAvailable(false)
	}
	for _, blockId := range secondWorkerDisabledBlocks {
		blockRegistry2.GetAvailableBlocks()[blockId].SetAvailable(false)
	}

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: testPipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: "test-block-first-slug",
			Input: map[string]interface{}{
				"url": imageUrl,
			},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server1,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	completedProcessing11 := <-notificationChannel1
	suite.Equal(completedProcessing11.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing11.GetStatus())
	suite.Nil(completedProcessing11.GetError())

	processingRegistry1Processing := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry1Processing)
	suite.Equal(imageContent.String(), completedProcessing11.GetOutput().GetValue().String())
	suite.Equal(completedProcessing11.GetId(), processingRegistry1Processing.GetId())

	transferredProcessing12 := <-notificationChannel1
	suite.Equal(transferredProcessing12.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusTransferred, transferredProcessing12.GetStatus())
	suite.Nil(transferredProcessing12.GetError())

	processingRegistry12Processing := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry1Processing)
	suite.Equal(transferredProcessing12.GetId(), processingRegistry12Processing.GetId())

	completedProcessing21 := <-notificationChannel2
	suite.Equal(completedProcessing21.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing21.GetStatus())
	processingRegistry2Processing1 := processingRegistry2.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry2Processing1)
	suite.Equal(completedProcessing21.GetId(), processingRegistry2Processing1.GetId())
}

func (suite *FunctionalTestSuite) TestTwoWorkersPipelineProcessingWorker1HasNoSecondBlockWorker2HasAllBlocks() {
	// Given
	imageWidth, imageHeight := 100, 100
	testPipelineSlug := "test-two-http-blocks"
	firstWorkerDisabledBlocks := []string{"image_add_text"}
	secondWorkerDisabledBlocks := []string{}
	imageContent := suite.GetPNGImageBuffer(imageWidth, imageHeight)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(imageContent.Bytes())
	}))
	suite.httpTestServers = append(suite.httpTestServers, server)
	imageUrl := server.URL

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(`{
			"slug": "%s",
			"title": "Test two Blocks",
			"description": "First Block downloads image and second block adds text to it",
			"blocks": [
				{
					"id": "http_request",
					"slug": "test-block-first-slug",
					"description": "Download Image from provided URL",
					"input": {
						"url": "%s"
					}
				},
				{
					"id": "image_add_text",
					"slug": "test-block-second-slug",
					"description": "Add text to downloaded image",
					"input_config": {
						"property": {
							"image": {
								"origin": "test-block-first-slug"
							}
						}
					},
					"input": {
						"text": "Hello, world!",
						"font_size": 50,
						"font_color": "#000000"
					}
				}
			]
		}`,
			testPipelineSlug,
			imageUrl,
		),
	)

	server1, worker1, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server1.GetPipelineRegistry().Add(pipeline)
	server2, worker2, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server2.GetPipelineRegistry().Add(pipeline)

	workerRegistry1 := server1.GetWorkerRegistry()
	workerRegistry2 := server2.GetWorkerRegistry()
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	notificationChannel1 := make(chan interfaces.Processing)
	notificationChannel2 := make(chan interfaces.Processing)

	processingRegistry1 := server1.GetProcessingRegistry()
	processingRegistry2 := server2.GetProcessingRegistry()

	processingRegistry1.SetNotificationChannel(notificationChannel1)
	processingRegistry2.SetNotificationChannel(notificationChannel2)

	blockRegistry1 := server1.GetBlockRegistry()
	blockRegistry2 := server2.GetBlockRegistry()

	for _, blockId := range firstWorkerDisabledBlocks {
		blockRegistry1.GetAvailableBlocks()[blockId].SetAvailable(false)
	}
	for _, blockId := range secondWorkerDisabledBlocks {
		blockRegistry2.GetAvailableBlocks()[blockId].SetAvailable(false)
	}

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: testPipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: "test-block-first-slug",
			Input: map[string]interface{}{
				"url": imageUrl,
			},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server1,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	completedProcessing11 := <-notificationChannel1
	suite.Equal(completedProcessing11.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing11.GetStatus())
	suite.Nil(completedProcessing11.GetError())

	processingRegistry1Processing := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry1Processing)
	suite.Equal(imageContent.String(), completedProcessing11.GetOutput().GetValue().String())
	suite.Equal(completedProcessing11.GetId(), processingRegistry1Processing.GetId())

	transferredProcessing12 := <-notificationChannel1
	suite.Equal(transferredProcessing12.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusTransferred, transferredProcessing12.GetStatus())
	suite.Nil(transferredProcessing12.GetError())

	processingRegistry12Processing := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry1Processing)
	suite.Equal(transferredProcessing12.GetId(), processingRegistry12Processing.GetId())

	completedProcessing21 := <-notificationChannel2
	suite.Equal(completedProcessing21.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing21.GetStatus())
	processingRegistry2Processing1 := processingRegistry2.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processingRegistry2Processing1)
	suite.Equal(completedProcessing21.GetId(), processingRegistry2Processing1.GetId())
}

func (suite *FunctionalTestSuite) TestTwoWorkersResumeProcessing2Times() {
	// Given
	imageWidth, imageHeight := 100, 100
	testPipelineSlug := "test-two-http-blocks"
	firstWorkerDisabledBlocks := []string{"image_add_text", "image_blur"}
	secondWorkerDisabledBlocks := []string{"http_request", "image_resize"}
	imageContent := suite.GetPNGImageBuffer(imageWidth, imageHeight)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(imageContent.Bytes())
	}))
	suite.httpTestServers = append(suite.httpTestServers, server)
	imageUrl := server.URL

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(`{
			"slug": "%s",
			"title": "Test two Blocks",
			"description": "First Block downloads image and second block adds text to it",
			"blocks": [
				{
					"id": "http_request",
					"slug": "test-block-first-slug",
					"description": "Download Image from provided URL",
					"input": {
						"url": "%s"
					}
				},
				{
					"id": "image_add_text",
					"slug": "test-block-second-slug",
					"description": "Add text to downloaded image",
					"input_config": {
						"property": {
							"image": {
								"origin": "test-block-first-slug"
							}
						}
					},
					"input": {
						"text": "Hello, world!",
						"font_size": 50,
						"font_color": "#000000"
					}
				},
				{
					"id": "image_resize",
					"slug": "test-block-third-slug",
					"description": "Resize image with text to 50x50",
					"input_config": {
						"property": {
							"image": {
								"origin": "test-block-second-slug"
							}
						}
					},
					"input": {
						"width": 50,
						"height": 50
					}
				},
				{
					"id": "image_blur",
					"slug": "test-block-fourth-slug",
					"description": "Blur image with sigma 1.0",
					"input_config": {
						"property": {
							"image": {
								"origin": "test-block-third-slug"
							}
						}
					},
					"input": {
						"sigma": 1.0
					}
				}
			]
		}`,
			testPipelineSlug,
			imageUrl,
		),
	)

	server1, worker1, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server1.GetPipelineRegistry().Add(pipeline)
	server2, worker2, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	server2.GetPipelineRegistry().Add(pipeline)

	workerRegistry1 := server1.GetWorkerRegistry()
	workerRegistry2 := server2.GetWorkerRegistry()
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	notificationChannel1 := make(chan interfaces.Processing)
	notificationChannel2 := make(chan interfaces.Processing)

	processingRegistry1 := server1.GetProcessingRegistry()
	processingRegistry2 := server2.GetProcessingRegistry()

	processingRegistry1.SetNotificationChannel(notificationChannel1)
	processingRegistry2.SetNotificationChannel(notificationChannel2)

	blockRegistry1 := server1.GetBlockRegistry()
	blockRegistry2 := server2.GetBlockRegistry()

	for _, blockId := range firstWorkerDisabledBlocks {
		blockRegistry1.GetAvailableBlocks()[blockId].SetAvailable(false)
	}
	for _, blockId := range secondWorkerDisabledBlocks {
		blockRegistry2.GetAvailableBlocks()[blockId].SetAvailable(false)
	}

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: testPipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: "test-block-first-slug",
			Input: map[string]interface{}{
				"url": imageUrl,
			},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server1,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	completedProcessing11 := <-notificationChannel1
	suite.Equal(completedProcessing11.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing11.GetStatus())
	suite.Nil(completedProcessing11.GetError())

	transferredProcessing12 := <-notificationChannel1
	suite.Equal(transferredProcessing12.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusTransferred, transferredProcessing12.GetStatus())
	suite.Nil(transferredProcessing12.GetError())

	completedProcessing22 := <-notificationChannel2
	suite.Equal(completedProcessing22.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing22.GetStatus())

	transferredProcessing23 := <-notificationChannel2
	suite.Equal(transferredProcessing23.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusTransferred, transferredProcessing23.GetStatus())

	completedProcessing13 := <-notificationChannel1
	suite.Equal(completedProcessing13.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing13.GetStatus())

	transferredProcessing14 := <-notificationChannel1
	suite.Equal(transferredProcessing14.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusTransferred, transferredProcessing14.GetStatus())

	completedProcessing24 := <-notificationChannel2
	suite.Equal(completedProcessing24.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing24.GetStatus())
}
