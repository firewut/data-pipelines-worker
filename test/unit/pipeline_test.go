package unit_test

import "data-pipelines-worker/types/dataclasses"

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
