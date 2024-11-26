package unit_test

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

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
	suite.NotNil(err, err)
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
		suite.GetWorkerRegistry(true),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		processingData,
		[]interfaces.Storage{types.NewLocalStorage("")},
	)

	// Then
	suite.NotNil(err, err)
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
	mockStorage := suite.NewMockLocalStorage(3)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(true),
		suite.GetBlockRegistry(),
		processingRegistry,
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
	suite.Equal(mockedResponse, processingOutput.GetValue()[0].String())

	<-mockStorage.GetCreatedFilesChan()
	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))
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
									"json_path": "$.action"
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
	mockStorage := suite.NewMockLocalStorage(3)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(true),
		suite.GetBlockRegistry(),
		processingRegistry,
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
	suite.Equal(mockedResponse, processing1Output.GetValue()[0].String())

	processing2 := <-notificationChannel
	suite.NotNil(processing2)
	suite.Equal(processingId, processing2.GetId())
	suite.Equal(interfaces.ProcessingStatusStopped, processing2.GetStatus())

	processing2Output := processing2.GetOutput()
	suite.NotNil(processing2Output)
	suite.True(processing2Output.GetStop())

	<-mockStorage.GetCreatedFilesChan()
	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))
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
									"json_path": "$.action"
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
	mockStorage := suite.NewMockLocalStorage(4)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(true),
		suite.GetBlockRegistry(),
		processingRegistry,
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
	suite.Equal(mockedResponse, processing1Output.GetValue()[0].String())

	processing2 := <-notificationChannel
	suite.NotNil(processing2)
	suite.Equal(processingId, processing2.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, processing2.GetStatus())

	processing2Output := processing2.GetOutput()
	suite.NotNil(processing2Output)
	suite.False(processing2Output.GetStop())

	<-mockStorage.GetCreatedFilesChan()
	<-mockStorage.GetCreatedFilesChan()
	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

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
									"json_path": "$"
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
	mockStorage := suite.NewMockLocalStorage(5)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(true),
		suite.GetBlockRegistry(),
		processingRegistry,
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
	suite.Equal(thirdBlockInput, processing1Output.GetValue()[0].String())

	processing2 := <-notificationChannel
	suite.NotNil(processing2)
	suite.Equal(processingId, processing2.GetId())
	suite.Equal(interfaces.ProcessingStatusStopped, processing2.GetStatus())

	processing2Output := processing2.GetOutput()
	suite.NotNil(processing2Output)
	suite.True(processing2Output.GetStop())

	<-mockStorage.GetCreatedFilesChan()
	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	// Third block must not be processed
	select {
	case <-time.After(500 * time.Millisecond):
	case sideProcessing := <-notificationChannel:
		suite.Fail(
			fmt.Sprintf(
				"Expected notification channel to be empty, but got a value: %s",
				sideProcessing.GetOutput().GetValue()[0].String(),
			),
		)
	}
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
									"json_path": "$"
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
	mockStorage := suite.NewMockLocalStorage(5)
	pipelineRegistry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(true),
		suite.GetBlockRegistry(),
		processingRegistry,
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
	suite.Equal(thirdBlockInput, processing1Output.GetValue()[0].String())

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
	suite.Equal(thirdBlockInput, processing1Output.GetValue()[0].String())

	<-mockStorage.GetCreatedFilesChan()
	<-mockStorage.GetCreatedFilesChan()
	<-mockStorage.GetCreatedFilesChan()
	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"log_id":"`)

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"log_id":"`)
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksOneProcess() {
	// Given
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	notificationChannel := make(chan interfaces.Processing, 2)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-first-slug",
		map[string]interface{}{
			"url": firstBlockInput,
		},
	)

	mockStorage := suite.NewMockLocalStorage(4)
	registry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
		},
	)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(true),
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

	secondBlockProcessing := <-notificationChannel
	suite.NotNil(secondBlockProcessing)
	suite.Equal(processingId, secondBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, secondBlockProcessing.GetStatus())

	firstBlockProcessingOutput := firstBlockProcessing.GetOutput()
	suite.NotNil(firstBlockProcessingOutput)
	suite.Equal(secondBlockInput, firstBlockProcessingOutput.GetValue()[0].String())

	secondBlockProcessingOutput := secondBlockProcessing.GetOutput()
	suite.NotNil(secondBlockProcessingOutput)
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessingOutput.GetValue()[0].String())

	<-mockStorage.GetCreatedFilesChan()
	<-mockStorage.GetCreatedFilesChan()
	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksOneProcessNStorages() {
	// Given
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 3)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 2)

	notificationChannel := make(chan interfaces.Processing, 2)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-first-slug",
		map[string]interface{}{
			"url": firstBlockInput,
		},
	)
	mockStorage1 := suite.NewMockLocalStorage(4)
	mockStorage2 := suite.NewMockLocalStorage(4)
	storages := []interfaces.Storage{
		types.NewLocalStorage(""),
		mockStorage1,
		mockStorage2,
	}
	registry.SetPipelineResultStorages(storages)

	// When
	processingId, err := pipeline.Process(
		suite.GetWorkerRegistry(true),
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

	secondBlockProcessing := <-notificationChannel
	suite.NotNil(secondBlockProcessing)
	suite.Equal(processingId, secondBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, secondBlockProcessing.GetStatus())

	firstBlockProcessingOutput := firstBlockProcessing.GetOutput()
	suite.NotNil(firstBlockProcessingOutput)
	suite.Equal(secondBlockInput, firstBlockProcessingOutput.GetValue()[0].String())

	secondBlockProcessingOutput := secondBlockProcessing.GetOutput()
	suite.NotNil(secondBlockProcessingOutput)
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessingOutput.GetValue()[0].String())

	// Check Storages
	createdFileBlock1 := <-mockStorage1.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(secondBlockInput, createdFileBlock1.data.String())

	createdFileBlock2 := <-mockStorage1.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock2)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock2.data.String())

	createdFileBlock1 = <-mockStorage2.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(secondBlockInput, createdFileBlock1.data.String())

	createdFileBlock2 = <-mockStorage2.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock2)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock2.data.String())

	pipelineLogFile1 := <-mockStorage1.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile1)

	pipelineStateFile1 := <-mockStorage1.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStateFile1)

	pipelineLogFile2 := <-mockStorage2.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile2)

	pipelineStateFile2 := <-mockStorage2.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStateFile2)

}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksResumeProcessOfSecondBlockInputPassed() {
	// Given
	processingId := uuid.New()
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	notificationChannel := make(chan interfaces.Processing, 2)
	processingRegistry := suite.GetProcessingRegistry(true)
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

	mockStorage := suite.NewMockLocalStorage(4)
	mockStorage.AddFile(
		mockStorage.NewStorageLocation(
			fmt.Sprintf(
				"%s/%s/%s/%s",
				"test-pipeline-slug-two-blocks",
				processingId.String(),
				"test-block-first-slug",
				"output_0.txt",
			),
		),
		bytes.NewBufferString(
			firstBlockInput,
		),
	)
	mockStorage.AddFile(
		mockStorage.NewStorageLocation(
			fmt.Sprintf(
				"%s/%s/%s/%s",
				"test-pipeline-slug-two-blocks",
				processingId.String(),
				"test-block-second-slug",
				"output_0.txt",
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
		suite.GetWorkerRegistry(true),
		suite.GetBlockRegistry(),
		processingRegistry,
		processingData,
		registry.GetPipelineResultStorages(),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	// Wait for Notification about processing completed
	secondBlockProcessing := <-notificationChannel
	suite.NotNil(secondBlockProcessing)
	suite.Equal(processingId, secondBlockProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, secondBlockProcessing.GetStatus())
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessing.GetOutput().GetValue()[0].String())

	createdFileBlock1 := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock1.data.String())

	createdFileBlock2 := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock2)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock1.data.String())

	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksResumeProcessOfSecondBlockInputMissing() {
	// Given
	processingId := uuid.New()
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.NewString(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)

	notificationChannel := make(chan interfaces.Processing, 2)
	processingRegistry := suite.GetProcessingRegistry(true)
	processingRegistry.SetNotificationChannel(notificationChannel)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-second-slug",
		map[string]interface{}{},
	)
	processingData.Pipeline.ProcessingID = processingId

	mockStorage := suite.NewMockLocalStorage(4)
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
		suite.GetWorkerRegistry(true),
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
	suite.Equal(mockedSecondBlockResponse, secondBlockProcessing.GetValue()[0].String())

	createdFileBlock1 := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock1.data.String())

	pipelineLogFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineLogFile)
	suite.Contains(pipelineLogFile.filePath, fmt.Sprintf("%s/log_", processingId.String()))
	suite.Contains(pipelineLogFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineLogFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineLogFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineLogFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))

	pipelineStatusFile := <-mockStorage.GetCreatedFilesChan()
	suite.NotEmpty(pipelineStatusFile)
	suite.Contains(pipelineStatusFile.filePath, fmt.Sprintf("%s/status_", processingId.String()))
	suite.Contains(pipelineStatusFile.data.String(), `"is_stopped":false`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_completed":true`)
	suite.Contains(pipelineStatusFile.data.String(), `"is_error":false`)
	suite.Contains(pipelineStatusFile.data.String(), fmt.Sprintf(`"id":"%s"`, processingId.String()))
}
