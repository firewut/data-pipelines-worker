package unit_test

import (
	"net/http"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/registries"
)

func (suite *UnitTestSuite) TestNewBlockRegistry() {
	blockRegistry := registries.NewBlockRegistry()

	registeredBlocks := blockRegistry.GetBlocks()
	suite.Greater(len(registeredBlocks), 0)

	availableBlocks := blockRegistry.GetAvailableBlocks()
	suite.Greater(len(availableBlocks), 0)

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

func (suite *UnitTestSuite) TestDetectBlocks() {
	// Config block Detection Conditions - mock URLs
	_config := config.GetConfig()
	mockUrls := make(map[string]string)

	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL("Mocked Response OK", http.StatusOK)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl

			mockUrls[blockId] = successUrl
		}
	}

	// Check config block detection condition points to mock URL
	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			suite.Equal(
				blockConfig.Detector.Conditions["url"].(string),
				mockUrls[blockId],
			)
		}
	}

	blockRegistry := registries.NewBlockRegistry()
	registeredBlocks := blockRegistry.GetBlocks()
	suite.Greater(len(registeredBlocks), 0)

	for _, block := range registeredBlocks {
		suite.True(block.IsAvailable())
	}
}
