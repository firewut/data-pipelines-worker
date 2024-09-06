package unit_test

import (
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/registries"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (suite *UnitTestSuite) TestNewPipelineRegistry() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	suite.NotNil(registry)
	suite.NotEmpty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryRegisterCorrect() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipeline, err := dataclasses.NewPipelineFromBytes(
		suite.GetTestPipelineDefinition(),
	)
	suite.Nil(err)
	suite.NotEmpty(pipeline.GetBlocks())

	registry.Register(pipeline)

	suite.NotEmpty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryRegisterErrorMissingRequiredProperty() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipeline, err := dataclasses.NewPipelineFromBytes(
		[]byte(`{
			"slug": "YT-CHANNEL-video-generation-invalid",
			"title": "Youtube Video generation Pipeline"
		}`),
	)
	suite.Nil(err)

	suite.Panics(func() {
		registry.Register(pipeline)
	})

	suite.Empty(registry.Get(pipeline.GetSlug()))
}

func (suite *UnitTestSuite) TestPipelineRegistryGet() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipeline, err := dataclasses.NewPipelineFromBytes(suite.GetTestPipelineDefinition())
	suite.Nil(err)

	registry.Register(pipeline)
	suite.NotEmpty(registry.GetAll())

	suite.NotEmpty(registry.Get("test"))
}

func (suite *UnitTestSuite) TestPipelineRegistryGetAll() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipeline, err := dataclasses.NewPipelineFromBytes(suite.GetTestPipelineDefinition())
	suite.Nil(err)

	registry.Register(pipeline)
	suite.NotEmpty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryDelete() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipeline, err := dataclasses.NewPipelineFromBytes(suite.GetTestPipelineDefinition())
	suite.Nil(err)

	registry.Register(pipeline)
	suite.NotEmpty(registry.Get("test"))

	registry.Delete("test")
	suite.Empty(registry.Get("test"))
}

func (suite *UnitTestSuite) TestPipelineRegistryLoadFromCatalogue() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipelineSlug := "YT-CHANNEL-video-generation-block-prompt"
	pipeline := registry.Get(pipelineSlug)

	suite.NotEmpty(pipeline)
	suite.Greater(len(pipeline.GetBlocks()), 0)

	implementedBlocks := map[string]string{
		"http_request": "openai_chat_completion",
	}

	for _, blockStructure := range pipeline.GetBlocks() {
		suite.NotEmpty(blockStructure)
		suite.Equal(pipeline, blockStructure.GetPipeline())

		for _, blockId := range implementedBlocks {
			if blockStructure.GetId() == blockId {
				blockData := blockStructure.GetBlock()
				suite.NotEmpty(blockData)
				suite.Equal(blockId, blockData.GetId())
			}
		}
	}
}

func (suite *UnitTestSuite) TestPipelineRegistryStartPipelineMissingPipelineTest() {
	// Given
	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK)

	_, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		"test-missing-pipeline-slug",
		"test-block-slug",
		successUrl,
	)

	// When
	processingId, err := registry.StartPipeline(processingData)

	// Then
	suite.NotNil(err)
	suite.Empty(processingId)
	suite.Equal(
		err,
		fmt.Errorf(
			"pipeline with slug %s not found",
			processingData.Pipeline.Slug,
		),
	)
}

func (suite *UnitTestSuite) TestPipelineRegistryStartPipelineMissingBlockTest() {
	// Given
	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK)

	_, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		"test-pipeline-slug",
		"test-missing-block-slug",
		successUrl,
	)

	// When
	processingId, err := registry.StartPipeline(processingData)

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

func (suite *UnitTestSuite) TestPipelineRegistryStartPipelineTest() {
	// Given
	mockedResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.New().String(),
	)
	successUrl := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK)
	_, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		"test-pipeline-slug",
		"test-block-slug",
		successUrl,
	)
	createdFilesChan := make(chan createdFile, 1)
	mockStorage := &mockLocalStorage{
		createdFilesChan: createdFilesChan,
	}
	registry.SetStorage(mockStorage)

	// When
	processingId, err := registry.StartPipeline(processingData)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	createdFile := <-createdFilesChan
	suite.NotEmpty(createdFile)
	suite.Equal(mockedResponse, createdFile.data.String())
}
