package unit_test

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
)

func (suite *UnitTestSuite) TestGetPipelinesConfigSchema() {
	suite.NotNil(suite._config.Pipeline.SchemaPtr)
}

func (suite *UnitTestSuite) TestNewPipelineErrorBrokenJSON() {
	_, err := dataclasses.NewPipelineFromBytes(
		[]byte(`{"slug": "test",
			"title": "Test Pipeline"`,
		),
	)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestNewPipelineCorrectJSON() {
	dataclasses.NewPipelineFromBytes([]byte(`{
		"slug": "test",
		"title": "Test Pipeline"
	}`))
}

func (suite *UnitTestSuite) TestPipelineProcessMissingBlock() {
	// Given
	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)

	pipeline, processingData, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-missing-block-slug",
		nil,
	)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		processingData,
		[]interfaces.Storage{types.NewLocalStorage("")},
	)

	// Then
	suite.NotNil(err)
	suite.Empty(processingId)
	suite.Equal(
		err,
		fmt.Errorf(
			"block with slug %s not found in pipeline %s",
			processingData.Block.Slug,
			processingData.Pipeline.Slug,
		),
	)
}

func (suite *UnitTestSuite) TestPipelineProcess() {
	// Given
	mockedResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	successUrl := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)
	pipeline, processingData, pipelineRegistry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	mockStorage := suite.NewMockLocalStorage(1)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		processingData,
		pipelineRegistry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notification about processing completed
	processing := <-notificationChannel
	suite.NotNil(processing)
	suite.Equal(processingId, processing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing.GetStatus())

	processingOutput := processing.GetOutput()
	suite.NotNil(processingOutput)
	suite.Equal(mockedResponse, processingOutput.GetValue().String())
}

func (suite *UnitTestSuite) TestPipelineProcessStopPipelineTrue() {
	// Given
	mockedResponse := `{"action": "declined"}`
	successUrl := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "test-pipeline-slug-two-blocks",
				"title": "Test Pipeline",
				"description": "Test Pipeline Description",
				"blocks": [
					{
						"id": "http_request",
						"slug": "test-block-first-slug",
						"description": "Request Local Resourse",
						"input": {
							"url": "%s"
						}
					},
					{
						"id": "stop_pipeline",
						"slug": "stop-pipeline-if-declined",
						"description": "Stop Pipeline if Declined",
						"input_config": {
							"property": {
								"data": {
									"origin": "test-block-first-slug",
									"jsonPath": "$.action"
								}
							}
						},
						"input": {
							"condition": "==",
							"value": "declined"
						}
					}
				]
			}`,
			successUrl,
		),
	)
	pipeline, processingData, pipelineRegistry := suite.RegisterTestPipelineAndInputForProcessing(
		pipeline,
		"test-pipeline-slug-two-blocks",
		"test-block-first-slug",
		nil,
	)
	mockStorage := suite.NewMockLocalStorage(1)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		processingData,
		pipelineRegistry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notifications
	processing1 := <-notificationChannel
	suite.NotNil(processing1)
	suite.Equal(processingId, processing1.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing1.GetStatus())

	processing1Output := processing1.GetOutput()
	suite.NotNil(processing1Output)
	suite.Equal(mockedResponse, processing1Output.GetValue().String())

	processing2 := <-notificationChannel
	suite.NotNil(processing2)
	suite.Equal(processingId, processing2.GetId())
	suite.Equal(interfaces.ProcessingStatusStopped, processing2.GetStatus())

	processing2Output := processing2.GetOutput()
	suite.NotNil(processing2Output)
	suite.True(processing2Output.GetStop())
}

func (suite *UnitTestSuite) TestPipelineProcessStopPipelineFalse() {
	// Given
	mockedResponse := `{"action": "accepted"}`
	successUrl := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "test-pipeline-slug-two-blocks",
				"title": "Test Pipeline",
				"description": "Test Pipeline Description",
				"blocks": [
					{
						"id": "http_request",
						"slug": "test-block-first-slug",
						"description": "Request Local Resourse",
						"input": {
							"url": "%s"
						}
					},
					{
						"id": "stop_pipeline",
						"slug": "stop-pipeline-if-declined",
						"description": "Stop Pipeline if Declined",
						"input_config": {
							"property": {
								"data": {
									"origin": "test-block-first-slug",
									"jsonPath": "$.action"
								}
							}
						},
						"input": {
							"condition": "==",
							"value": "declined"
						}
					}
				]
			}`,
			successUrl,
		),
	)
	pipeline, processingData, pipelineRegistry := suite.RegisterTestPipelineAndInputForProcessing(
		pipeline,
		"test-pipeline-slug-two-blocks",
		"test-block-first-slug",
		nil,
	)
	mockStorage := suite.NewMockLocalStorage(2)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		processingData,
		pipelineRegistry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notifications
	processing1 := <-notificationChannel
	suite.NotNil(processing1)
	suite.Equal(processingId, processing1.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing1.GetStatus())

	processing1Output := processing1.GetOutput()
	suite.NotNil(processing1Output)
	suite.Equal(mockedResponse, processing1Output.GetValue().String())

	processing2 := <-notificationChannel
	suite.NotNil(processing2)
	suite.Equal(processingId, processing2.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing2.GetStatus())

	processing2Output := processing2.GetOutput()
	suite.NotNil(processing2Output)
	suite.False(processing2Output.GetStop())
}

func (suite *UnitTestSuite) TestPipelineProcessStopPipelineTrueThreeBlocks() {
	// Given
	mockedResponse := `{"action": "accepted"}`
	thirdBlockInput := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(thirdBlockInput, http.StatusOK, 0)

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "test-pipeline-slug-three-blocks",
				"title": "Test Pipeline",
				"description": "Test Pipeline Description",
				"blocks": [
					{
						"id": "http_request",
						"slug": "test-block-first-slug",
						"description": "Request Local Resourse",
						"input": {
							"url": "%s"
						}
					},
					{
						"id": "stop_pipeline",
						"slug": "stop-pipeline-if-declined",
						"description": "Stop Pipeline if Declined",
						"input_config": {
							"property": {
								"data": {
									"origin": "test-block-first-slug",
									"jsonPath": "$"
								}
							}
						},
						"input": {
							"condition": "==",
							"value": "%s"
						}
					},
					{
						"id": "http_request",
						"slug": "test-block-third-slug",
						"description": "Request Result from First Block",
						"input_config": {
							"property": {
								"url": {
									"origin": "test-block-first-slug"
								}
							}
						}
					}
				]
			}`,
			firstBlockInput,
			thirdBlockInput,
		),
	)
	pipeline, processingData, pipelineRegistry := suite.RegisterTestPipelineAndInputForProcessing(
		pipeline,
		"test-pipeline-slug-three-blocks",
		"test-block-first-slug",
		nil,
	)
	mockStorage := suite.NewMockLocalStorage(3)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		processingData,
		pipelineRegistry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notifications
	processing1 := <-notificationChannel
	suite.NotNil(processing1)
	suite.Equal(processingId, processing1.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing1.GetStatus())

	processing1Output := processing1.GetOutput()
	suite.NotNil(processing1Output)
	suite.Equal(thirdBlockInput, processing1Output.GetValue().String())

	processing2 := <-notificationChannel
	suite.NotNil(processing2)
	suite.Equal(processingId, processing2.GetId())
	suite.Equal(interfaces.ProcessingStatusStopped, processing2.GetStatus())

	processing2Output := processing2.GetOutput()
	suite.NotNil(processing2Output)
	suite.True(processing2Output.GetStop())
}
func (suite *UnitTestSuite) TestPipelineProcessStopPipelineFalseThreeBlocks() {
	// Given
	mockedResponse := `{"action": "accepted"}`
	thirdBlockInput := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(thirdBlockInput, http.StatusOK, 0)

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "test-pipeline-slug-three-blocks",
				"title": "Test Pipeline",
				"description": "Test Pipeline Description",
				"blocks": [
					{
						"id": "http_request",
						"slug": "test-block-first-slug",
						"description": "Request Local Resourse",
						"input": {
							"url": "%s"
						}
					},
					{
						"id": "stop_pipeline",
						"slug": "stop-pipeline-if-declined",
						"description": "Stop Pipeline if Declined",
						"input_config": {
							"property": {
								"data": {
									"origin": "test-block-first-slug",
									"jsonPath": "$"
								}
							}
						},
						"input": {
							"condition": "!=",
							"value": "%s"
						}
					},
					{
						"id": "http_request",
						"slug": "test-block-third-slug",
						"description": "Request Result from First Block",
						"input_config": {
							"property": {
								"url": {
									"origin": "test-block-first-slug"
								}
							}
						}
					}
				]
			}`,
			firstBlockInput,
			thirdBlockInput,
		),
	)
	pipeline, processingData, pipelineRegistry := suite.RegisterTestPipelineAndInputForProcessing(
		pipeline,
		"test-pipeline-slug-three-blocks",
		"test-block-first-slug",
		nil,
	)
	mockStorage := suite.NewMockLocalStorage(3)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		processingData,
		pipelineRegistry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notifications
	processing1 := <-notificationChannel
	suite.NotNil(processing1)
	suite.Equal(processingId, processing1.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing1.GetStatus())

	processing1Output := processing1.GetOutput()
	suite.NotNil(processing1Output)
	suite.Equal(thirdBlockInput, processing1Output.GetValue().String())

	processing2 := <-notificationChannel
	suite.NotNil(processing2)
	suite.Equal(processingId, processing2.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing2.GetStatus())

	processing2Output := processing2.GetOutput()
	suite.NotNil(processing2Output)
	suite.False(processing2Output.GetStop())

	processing3 := <-notificationChannel
	suite.NotNil(processing3)
	suite.Equal(processingId, processing3.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing3.GetStatus())

	processing3Output := processing3.GetOutput()
	suite.NotNil(processing3Output)
	suite.Equal(thirdBlockInput, processing1Output.GetValue().String())
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksOneProcess() {
	// Given
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-first-slug",
		map[string]interface{}{
			"url": firstBlockInput,
		},
	)

	mockStorage := suite.NewMockLocalStorage(2)
	registry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		processingRegistry,
		processingData,
		registry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notification about processing completed
	firstBlockProcessing := <-notificationChannel
	suite.NotNil(firstBlockProcessing)
	suite.Equal(processingId, firstBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, firstBlockProcessing.GetStatus())

	firstBlockProcessingOutput := firstBlockProcessing.GetOutput()
	suite.NotNil(firstBlockProcessingOutput)
	suite.Equal(secondBlockInput, firstBlockProcessingOutput.GetValue().String())

	secondBlockProcessing := <-notificationChannel
	suite.NotNil(secondBlockProcessing)
	suite.Equal(processingId, secondBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, secondBlockProcessing.GetStatus())

	secondBlockProcessingOutput := secondBlockProcessing.GetOutput()
	suite.NotNil(secondBlockProcessingOutput)
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessingOutput.GetValue().String())
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksOneProcessNStorages() {
	// Given
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-first-slug",
		map[string]interface{}{
			"url": firstBlockInput,
		},
	)
	mockStorage1 := suite.NewMockLocalStorage(2)
	mockStorage2 := suite.NewMockLocalStorage(2)
	storages := []interfaces.Storage{
		types.NewLocalStorage(""),
		mockStorage1,
		mockStorage2,
	}
	registry.SetPipelineResultStorages(storages)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		processingRegistry,
		processingData,
		registry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notification about processing completed
	firstBlockProcessing := <-notificationChannel
	suite.NotNil(firstBlockProcessing)
	suite.Equal(processingId, firstBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, firstBlockProcessing.GetStatus())

	firstBlockProcessingOutput := firstBlockProcessing.GetOutput()
	suite.NotNil(firstBlockProcessingOutput)
	suite.Equal(secondBlockInput, firstBlockProcessingOutput.GetValue().String())

	secondBlockProcessing := <-notificationChannel
	suite.NotNil(secondBlockProcessing)
	suite.Equal(processingId, secondBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, secondBlockProcessing.GetStatus())

	secondBlockProcessingOutput := secondBlockProcessing.GetOutput()
	suite.NotNil(secondBlockProcessingOutput)
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessingOutput.GetValue().String())

	// Check Storages
	createdFileBlock1 := <-mockStorage1.createdFilesChan
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(secondBlockInput, createdFileBlock1.data.String())

	createdFileBlock2 := <-mockStorage1.createdFilesChan
	suite.NotEmpty(createdFileBlock2)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock2.data.String())

	createdFileBlock1 = <-mockStorage2.createdFilesChan
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(secondBlockInput, createdFileBlock1.data.String())

	createdFileBlock2 = <-mockStorage2.createdFilesChan
	suite.NotEmpty(createdFileBlock2)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock2.data.String())
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksResumeProcessOfSecondBlockInputPassed() {
	// Given
	processingId := uuid.New()
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)

	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-second-slug",
		map[string]interface{}{
			"url": secondBlockInput,
		},
	)
	processingData.Pipeline.ProcessingID = processingId

	mockStorage := suite.NewMockLocalStorage(2)
	mockStorage.AddFile(
		mockStorage.NewStorageLocation(
			fmt.Sprintf(
				"%s/%s/%s/%s",
				"test-pipeline-slug-two-blocks",
				processingId.String(),
				"test-block-first-slug",
				"output_1.txt",
			),
		),
		bytes.NewBufferString(
			"this value must be ignored since URL is passed in Request",
		),
	)

	registry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		processingRegistry,
		processingData,
		registry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notification about processing completed
	firstBlockProcessing := <-notificationChannel
	suite.NotNil(firstBlockProcessing)
	suite.Equal(processingId, firstBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, firstBlockProcessing.GetStatus())

	secondBlockProcessing := firstBlockProcessing.GetOutput()
	suite.NotNil(secondBlockProcessing)
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessing.GetValue().String())

	createdFileBlock1 := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock1.data.String())
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksResumeProcessOfSecondBlockInputMissing() {
	// Given
	processingId := uuid.New()
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)

	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-second-slug",
		map[string]interface{}{},
	)
	processingData.Pipeline.ProcessingID = processingId

	mockStorage := suite.NewMockLocalStorage(2)
	mockStorage.AddFile(
		mockStorage.NewStorageLocation(
			fmt.Sprintf(
				"%s/%s/%s/%s",
				"test-pipeline-slug-two-blocks",
				processingId.String(),
				"test-block-first-slug",
				"output_1.txt",
			),
		),
		bytes.NewBufferString(secondBlockInput),
	)

	registry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		processingRegistry,
		processingData,
		registry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notification about processing completed
	firstBlockProcessing := <-notificationChannel
	suite.NotNil(firstBlockProcessing)
	suite.Equal(processingId, firstBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, firstBlockProcessing.GetStatus())

	secondBlockProcessing := firstBlockProcessing.GetOutput()
	suite.NotNil(secondBlockProcessing)
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessing.GetValue().String())

	createdFileBlock1 := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock1.data.String())
}
