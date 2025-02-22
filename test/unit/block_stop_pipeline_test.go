package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockStopPipeline() {
	block := blocks.NewBlockStopPipeline()

	suite.Equal("stop_pipeline", block.GetId())
	suite.Equal("Pipeline Stop", block.GetName())
	suite.Equal("Stop the pipeline if a condition is met", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Empty(blockConfig.Value)
	suite.Empty(blockConfig.Condition)
	suite.Empty(blockConfig.Data)
}

func (suite *UnitTestSuite) TestBlockStopPipelineValidateSchemaOk() {
	block := blocks.NewBlockStopPipeline()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockStopPipelineValidateSchemaFail() {
	block := blocks.NewBlockStopPipeline()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockStopPipelineProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockStopPipeline()
	data := &dataclasses.BlockData{
		Id:   "stop_pipeline",
		Slug: "stop-pipeline-slug",
		Input: map[string]interface{}{
			"data":      nil,
			"condition": nil,
			"value":     nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorStopPipeline(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockStopPipelineProcessSuccess() {
	// Given
	type testCase struct {
		data      string
		value     string
		condition string
		stop      bool
	}
	testCases := []testCase{
		{
			data:      "data",
			value:     "value",
			condition: "==",
			stop:      false,
		},
		{
			data:      "data",
			value:     "data",
			condition: "==",
			stop:      true,
		},
		{
			data:      "data",
			value:     "datA",
			condition: "!=",
			stop:      true,
		},
	}

	for _, tc := range testCases {

		block := blocks.NewBlockStopPipeline()
		data := &dataclasses.BlockData{
			Id:   "stop_pipeline",
			Slug: "stop-pipeline-slug",
			Input: map[string]interface{}{
				"data":      tc.data,
				"condition": tc.condition,
				"value":     tc.value,
			},
		}
		data.SetBlock(block)

		// When
		result, stop, _, _, _, err := block.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorStopPipeline(),
			data,
		)

		// Then
		suite.Empty(result)
		suite.Equal(stop, tc.stop, tc.data, tc.value, tc.condition)
		suite.Nil(err)
	}
}
