package unit_test

import (
	"fmt"
	"net/http"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"

	"github.com/google/uuid"
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
	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK)

	pipeline, processingData, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-missing-block-slug",
		nil,
	)

	// When
	processingId, err := pipeline.Process(
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
		uuid.New().String(),
	)
	successUrl := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK)
	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	createdFilesChan := make(chan createdFile, 1)
	mockStorage := &mockLocalStorage{
		createdFilesChan: createdFilesChan,
	}
	registry.SetPipelineResultStorages(
		[]interfaces.Storage{
			mockStorage,
			// types.NewMINIOStorage(),
		},
	)

	// When
	processingId, err := pipeline.Process(processingData, registry.GetPipelineResultStorages())

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	createdFile := <-createdFilesChan
	suite.NotEmpty(createdFile)
	suite.Equal(mockedResponse, createdFile.data.String())
}

func (suite *UnitTestSuite) TestPipelineProcessTwoBlocksOneProcess() {
	// Given
	mockedSecondBlockResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.New().String(),
	)
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK)

	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks("NOT URL AT ALL"),
		"test-pipeline-slug-two-blocks",
		"test-block-first-slug",
		map[string]interface{}{
			"url": firstBlockInput,
		},
	)
	createdFilesChan := make(chan createdFile, 2)
	mockStorage := &mockLocalStorage{
		createdFilesChan: createdFilesChan,
	}
	registry.SetPipelineResultStorages(
		[]interfaces.Storage{mockStorage},
	)

	// When
	processingId, err := pipeline.Process(processingData, registry.GetPipelineResultStorages())

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	createdFileBlock1 := <-createdFilesChan
	suite.NotEmpty(createdFileBlock1)
	suite.Equal(secondBlockInput, createdFileBlock1.data.String())

	createdFileBlock2 := <-createdFilesChan
	suite.NotEmpty(createdFileBlock2)
	suite.Equal(mockedSecondBlockResponse, createdFileBlock2.data.String())
}
