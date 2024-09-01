package unit_test

import "data-pipelines-worker/types/registries"

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
