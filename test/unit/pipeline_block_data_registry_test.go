package unit_test

import (
	"bytes"
	"fmt"
	"path/filepath"

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
	outputIndex := 5

	fileNamePattern := filepath.Join(
		pipelineSlug,
		processingId.String(),
		blockSlug,
		fmt.Sprintf(
			"output_%d.txt",
			outputIndex,
		),
	)
	filePathPattern := filepath.Join(
		storages[0].GetStorageDirectory(),
		fileNamePattern,
	)

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
		blockSlug, outputIndex, bytes.NewBufferString(outputString),
	)

	// Then
	suite.NotEmpty(saveOutputResults)
	for _, saveOutputResult := range saveOutputResults {
		suite.NotNil(saveOutputResult.StorageLocation)

		suite.Equal(
			"local",
			saveOutputResult.StorageLocation.GetStorage().GetStorageName(),
		)
		suite.Nil(saveOutputResult.Error)

		suite.Equal(
			fileNamePattern,
			saveOutputResult.StorageLocation.GetFileName(),
		)
		suite.Equal(
			filePathPattern,
			saveOutputResult.StorageLocation.GetFilePath(),
		)

		defer saveOutputResult.StorageLocation.Delete()

		fileContent, err := saveOutputResult.StorageLocation.GetObjectBytes()
		suite.Nil(err)
		suite.Equal(outputString, fileContent.String())

		// Read file content using Registry Methods
		filesContent := pipelineBlockDataRegistry.LoadOutput(blockSlug)
		suite.NotEmpty(filesContent)
		suite.Equal(outputString, filesContent[0].String())

		filesContent = pipelineBlockDataRegistry.Get(blockSlug)
		suite.NotEmpty(filesContent)
		suite.Equal(outputString, filesContent[0].String())
	}
}

func (suite *UnitTestSuite) TestNewPipelineBlockDataRegistrySaveOutputMinioStorage() {
	// Given
	processingId := uuid.New()
	pipelineSlug := "test-pipeline-slug"
	blockSlug := "test-block-slug"
	storages := []interfaces.Storage{
		types.NewMINIOStorage(),
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
		suite.Equal("minio", saveOutputResult.StorageLocation.GetStorageName())
		suite.NotEmpty(saveOutputResult.StorageLocation.GetFilePath())
		suite.Nil(saveOutputResult.Error)

		defer saveOutputResult.StorageLocation.Delete()

		// Read file content using Registry Methods
		filesContent := pipelineBlockDataRegistry.LoadOutput(blockSlug)
		suite.NotEmpty(filesContent)
		suite.Equal(outputString, filesContent[0].String())

		filesContent = pipelineBlockDataRegistry.Get(blockSlug)
		suite.NotEmpty(filesContent)
		suite.Equal(outputString, filesContent[0].String())
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
		suite.Equal("mock-no-space-left", saveOutputResult.StorageLocation.GetStorageName())
		suite.NotNil(saveOutputResult.Error)
	}
}
