package unit_test

import (
	"net/http"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/registries"
)

func (suite *UnitTestSuite) TestNewBlockRegistry() {
	// Given
	blockRegistry := registries.NewBlockRegistry()
	defer blockRegistry.Shutdown(
		suite.GetShutDownContext(
			time.Second,
		),
	)

	// When
	registeredBlocks := blockRegistry.GetAll()
	suite.Greater(len(registeredBlocks), 0)

	availableBlocks := blockRegistry.GetAvailableBlocks()
	suite.Greater(len(availableBlocks), 0)

	// Then
	for _, block := range registeredBlocks {
		suite.NotEmpty(block.GetId())
		suite.NotEmpty(block.GetName())
		suite.NotEmpty(block.GetDescription())
		suite.NotEmpty(block.GetSchemaString())
		suite.NotNil(block.GetSchema())
	}
}

func (suite *UnitTestSuite) TestGetBlockRegistry() {
	blockRegistry := registries.GetBlockRegistry()
	suite.NotEmpty(blockRegistry)
}

func (suite *UnitTestSuite) TestBlockRegistryDetectBlocks() {
	// Given
	_config := config.GetConfig()
	mockUrls := make(map[string]string)

	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL("Mocked Response OK", http.StatusOK, 0)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl

			mockUrls[blockId] = successUrl
		}
	}
	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			suite.Equal(
				blockConfig.Detector.Conditions["url"].(string),
				mockUrls[blockId],
			)
		}
	}

	// When
	blockRegistry := registries.NewBlockRegistry()

	// Then
	defer blockRegistry.Shutdown(
		suite.GetShutDownContext(time.Second),
	)
	registeredBlocks := blockRegistry.GetAll()
	suite.Greater(len(registeredBlocks), 0)

	for _, block := range registeredBlocks {
		suite.True(block.IsAvailable())
	}
}

func (suite *UnitTestSuite) TestBlockRegistryShutdown() {
	// Given
	_config := config.GetConfig()
	mockUrls := make(map[string]string)

	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL("Mocked Response OK", http.StatusOK, 0)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl

			mockUrls[blockId] = successUrl
		}
	}
	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			suite.Equal(
				blockConfig.Detector.Conditions["url"].(string),
				mockUrls[blockId],
			)
		}
	}

	blockRegistry := registries.NewBlockRegistry()
	registeredBlocks := blockRegistry.GetAll()
	suite.Greater(len(registeredBlocks), 0)

	for _, block := range registeredBlocks {
		suite.True(block.IsAvailable())
	}

	// When
	blockRegistry.Shutdown(
		suite.GetShutDownContext(time.Second),
	)

	// Then
	for _, block := range registeredBlocks {
		suite.False(block.IsAvailable())
	}
}
