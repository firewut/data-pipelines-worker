package unit_test

import (
	"fmt"
	"net/http"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/dataclasses"

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

func (suite *UnitTestSuite) TestPipelineStartProcessingMissingBlock() {
	// Given
	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK)

	pipeline, processingData, _ := suite.RegisterTestPipelineAndInputForProcessing(
		"test-pipeline-slug",
		"test-missing-block-slug",
		successUrl,
	)

	// When
	processingId, err := pipeline.StartProcessing(
		processingData,
		types.NewLocalStorage(),
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

func (suite *UnitTestSuite) TestPipelineStartProcessing() {
	// Given
	mockedResponse := fmt.Sprintf(
		"Hello, world! Mocked value is %s",
		uuid.New().String(),
	)
	successUrl := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK)
	pipeline, processingData, registry := suite.RegisterTestPipelineAndInputForProcessing(
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
	processingId, err := pipeline.StartProcessing(
		processingData,
		mockStorage,
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(processingId)

	createdFile := <-createdFilesChan
	suite.NotEmpty(createdFile)
	suite.Equal(mockedResponse, createdFile.data.String())
}
