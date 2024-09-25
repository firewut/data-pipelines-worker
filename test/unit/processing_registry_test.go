package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

func (suite *UnitTestSuite) TestNewProcessingRegistry() {
	registry := registries.NewProcessingRegistry()

	suite.NotNil(registry)
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestGetProcessingRegistry() {
	// Given
	cases := [][]bool{
		{false, false, true}, // Expect same instance
		{true, false, false}, // Expect the same instance as the one created
		{false, true, false}, // Expect different instances
		{true, true, false},  // Expect different instances
	}

	// When
	for _, c := range cases {
		processingRegistry1 := registries.GetProcessingRegistry(c[0])
		processingRegistry2 := registries.GetProcessingRegistry(c[1])

		// Then
		suite.Equal(processingRegistry1 == processingRegistry2, c[2])
	}
}

func (suite *UnitTestSuite) TestProcessingRegistryAdd() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)

	// When
	registry.Add(processing)

	// Then
	suite.NotEmpty(registry.GetAll())
	suite.Equal(1, len(registry.GetAll()))
	suite.Equal(processing, registry.Get(processingId.String()))
}

func (suite *UnitTestSuite) TestProcessingRegistryGetAll() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)

	// When
	processings := registry.GetAll()

	// Then
	suite.NotEmpty(processings)
	suite.Equal(1, len(processings))
	suite.Equal(processing, processings[processing.GetId().String()])
}

func (suite *UnitTestSuite) TestProcessingRegistryGet() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)

	// When
	result := registry.Get(processingId.String())

	// Then
	suite.NotNil(result)
	suite.Equal(processing, result)
}

func (suite *UnitTestSuite) TestProcessingRegistryDelete() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	// When
	registry.Delete(processingId.String())

	// Then
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestProcessingRegistryDeleteAll() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	// When
	registry.DeleteAll()

	// Then
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestProcessingRegistryShutdownEmptyRegistry() {
	// Given
	registry := registries.NewProcessingRegistry()
	ctx := suite.GetShutDownContext(time.Second)

	// When
	err := registry.Shutdown(ctx)

	// Then
	suite.Nil(err)
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestProcessingRegistryStartProcessingById() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": successUrl,
		},
	)

	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	go registry.StartProcessingById(processingId)

	// When
	completedProcessing := <-registry.GetProcessingCompletedChannel()

	// Then
	suite.NotEmpty(completedProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing.GetStatus())
}

func (suite *UnitTestSuite) TestProcessingRegistryStartProcessing() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": successUrl,
		},
	)

	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	// registry.Add(processing)
	suite.Empty(registry.GetAll())

	go registry.StartProcessing(processing)

	// When
	completedProcessing := <-registry.GetProcessingCompletedChannel()

	// Then
	suite.NotEmpty(registry.GetAll())
	suite.NotEmpty(completedProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing.GetStatus())
}

func (suite *UnitTestSuite) TestProcessingRegistryStartProcessingTwoBlocks() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	processingCompletedChannel := registry.GetProcessingCompletedChannel()
	block := blocks.NewBlockHTTP()

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, time.Millisecond)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks(firstBlockInput),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": firstBlockInput,
		},
	)

	processing1 := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-first-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	processing2 := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-second-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing2.GetStatus())
	suite.Empty(registry.GetAll())

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		registry.StartProcessing(processing1)
		wg.Done()

		registry.StartProcessing(processing2)
		wg.Done()
	}()
	wg.Wait()

	// When
	completedProcessing1 := <-processingCompletedChannel
	completedProcessing2 := <-processingCompletedChannel

	// Then
	suite.NotEmpty(completedProcessing1.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing1.GetStatus())

	suite.NotEmpty(completedProcessing2.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing2.GetStatus())
}

func (suite *UnitTestSuite) TestProcessingRegistryShutdownCompletedProcessing() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	processDelay := time.Millisecond * 10
	shutdownTriggerDelay := processDelay + time.Millisecond*5
	shutdownDelay := time.Millisecond * 20

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, processDelay)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": successUrl,
		},
	)
	ctx := suite.GetShutDownContext(shutdownDelay)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	go registry.StartProcessingById(processingId)

	time.Sleep(shutdownTriggerDelay)

	// When
	err := registry.Shutdown(ctx)

	// Then
	suite.Nil(err)
	suite.Equal(interfaces.ProcessingStatusCompleted, processing.GetStatus())
}

func (suite *UnitTestSuite) TestProcessingRegistryShutdownRunningProcessing() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	processDelay := time.Hour
	shutdownTriggerDelay := time.Millisecond * 10
	shutdownDelay := time.Millisecond * 10

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, processDelay)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": successUrl,
		},
	)
	ctx := suite.GetShutDownContext(shutdownDelay)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	go registry.StartProcessingById(processingId)

	time.Sleep(shutdownTriggerDelay)

	// When
	err := registry.Shutdown(ctx)

	// Then
	suite.NotNil(err)
	suite.Equal(context.DeadlineExceeded, err)
}
