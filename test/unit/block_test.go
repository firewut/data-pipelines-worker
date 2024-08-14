package unit_test

import (
	"data-pipelines-worker/types/blocks"
)

func (suite *UnitTestSuite) TestDetectBlocks() {
	blocks := blocks.DetectBlocks()
	suite.Len(blocks, 0)
}
