package unit_test

import (
	"data-pipelines-worker/types/blocks"
)

func (suite *UnitTestSuite) TestDetectBlocks() {
	blocks := blocks.DetectBlocks()
	suite.Greater(len(blocks), 0)

	for _, block := range blocks {
		suite.NotEmpty(block.GetId())
		suite.NotEmpty(block.GetName())
		suite.NotEmpty(block.GetDescription())
		suite.NotEmpty(block.GetSchemaString())
		suite.NotNil(block.GetSchema())
	}
}
