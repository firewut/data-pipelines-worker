package unit_test

import (
	"data-pipelines-worker/types"
)

func (suite *UnitTestSuite) TestGetPipelinesConfigSchema() {
	config := types.GetConfig()

	suite.NotNil(config.Pipeline.SchemaPtr)
}

func (suite *UnitTestSuite) TestNewPipelineErrorBrokenJSON() {
	suite.Panics(func() {
		types.NewPipeline([]byte(`{"slug": "test",
			"title": "Test Pipeline"`))
	})
}

func (suite *UnitTestSuite) TestNewPipelineCorrectJSON() {
	types.NewPipeline([]byte(`{
		"slug": "test",
		"title": "Test Pipeline"
	}`))
}
