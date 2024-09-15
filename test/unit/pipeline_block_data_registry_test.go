package unit_test

import (
	"bytes"
	"os"

	"github.com/google/uuid"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

func (suite *UnitTestSuite) TestNewPipelineBlockDataRegistry() {
	// Given
	processingId := uuid.New()
	pipelineSlug := "test-pipeline-slug"
	storages := []interfaces.Storage{
		suite.NewMockLocalStorage(0),
	}

	// When
	pipelineBlockDataRegistry := registries.NewPipelineBlockDataRegistry(
		processingId,
		pipelineSlug,
		storages,
	)

	// Then
	suite.NotNil(pipelineBlockDataRegistry)
	suite.Equal(
		processingId,
		pipelineBlockDataRegistry.GetProcessingId(),
	)
	suite.Equal(
		pipelineSlug,
		pipelineBlockDataRegistry.GetPipelineSlug(),
	)
	suite.Equal(
		storages,
		pipelineBlockDataRegistry.GetStorages(),
	)
}

func (suite *UnitTestSuite) TestNewPipelineBlockDataRegistrySetStorages() {
	// Given
	processingId := uuid.New()
	pipelineSlug := "test-pipeline-slug"
	storages := []interfaces.Storage{
		suite.NewMockLocalStorage(0),
	}
	newStorages := []interfaces.Storage{
		suite.NewMockLocalStorage(1),
	}

	pipelineBlockDataRegistry := registries.NewPipelineBlockDataRegistry(
		processingId,
		pipelineSlug,
		storages,
	)
	suite.NotNil(pipelineBlockDataRegistry)
	suite.Equal(
		processingId,
		pipelineBlockDataRegistry.GetProcessingId(),
	)
	suite.Equal(
		pipelineSlug,
		pipelineBlockDataRegistry.GetPipelineSlug(),
	)

	// When
	pipelineBlockDataRegistry.SetStorages(newStorages)

	// Then
	suite.Equal(
		newStorages,
		pipelineBlockDataRegistry.GetStorages(),
	)
}

func (suite *UnitTestSuite) TestNewPipelineBlockDataRegistrySaveOutputLocalStorage() {
	// Given
	processingId := uuid.New()
	pipelineSlug := "test-pipeline-slug"
	blockSlug := "test-block-slug"
	storages := []interfaces.Storage{
		types.NewLocalStorage(""),
	}
	outputString := "Hello, world!"

	pipelineBlockDataRegistry := registries.NewPipelineBlockDataRegistry(
		processingId,
		pipelineSlug,
		storages,
	)
	suite.NotNil(pipelineBlockDataRegistry)
	suite.Equal(
		processingId,
		pipelineBlockDataRegistry.GetProcessingId(),
	)
	suite.Equal(
		pipelineSlug,
		pipelineBlockDataRegistry.GetPipelineSlug(),
	)

	// When
	saveOutputResults := pipelineBlockDataRegistry.SaveOutput(
		blockSlug, 6, bytes.NewBufferString(outputString),
	)

	// Then
	for _, saveOutputResult := range saveOutputResults {
		suite.Equal("local", saveOutputResult.Storage.GetStorageName())
		suite.NotEmpty(saveOutputResult.Path)
		suite.Nil(saveOutputResult.Error)

		defer os.Remove(saveOutputResult.Path)
		// Read file content
		fileContent, err := saveOutputResult.Storage.GetObjectBytes("", saveOutputResult.Path)
		suite.Nil(err)
		suite.Equal(outputString, fileContent.String())
	}
}

func (suite *UnitTestSuite) TestNewPipelineBlockDataRegistrySaveOutputNoSpaceOnDeviceLeft() {
	// Given
	processingId := uuid.New()
	pipelineSlug := "test-pipeline-slug"
	blockSlug := "test-block-slug"
	storages := []interfaces.Storage{
		&noSpaceLeftLocalStorage{},
	}
	outputString := "Hello, world!"

	pipelineBlockDataRegistry := registries.NewPipelineBlockDataRegistry(
		processingId,
		pipelineSlug,
		storages,
	)
	suite.NotNil(pipelineBlockDataRegistry)
	suite.Equal(
		processingId,
		pipelineBlockDataRegistry.GetProcessingId(),
	)
	suite.Equal(
		pipelineSlug,
		pipelineBlockDataRegistry.GetPipelineSlug(),
	)

	// When
	saveOutputResults := pipelineBlockDataRegistry.SaveOutput(
		blockSlug, 6, bytes.NewBufferString(outputString),
	)

	// Then
	for _, saveOutputResult := range saveOutputResults {
		suite.Equal("mock-no-space-left", saveOutputResult.Storage.GetStorageName())
		suite.Empty(saveOutputResult.Path)
		suite.NotNil(saveOutputResult.Error)
	}
}
