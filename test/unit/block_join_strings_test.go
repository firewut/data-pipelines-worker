package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockJoinStrings() {
	block := blocks.NewBlockJoinStrings()

	suite.Equal("join_strings", block.GetId())
	suite.Equal("Join Strings", block.GetName())
	suite.Equal("Join multiple strings into one", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("", blockConfig.Separator)
}

func (suite *UnitTestSuite) TestBlockJoinStringsValidateSchemaOk() {
	block := blocks.NewBlockJoinStrings()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockJoinStringsValidateSchemaFail() {
	block := blocks.NewBlockJoinStrings()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockJoinStringsProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockJoinStrings()
	data := &dataclasses.BlockData{
		Id:   "join_strings",
		Slug: "string-from-list-of-strings",
		Input: map[string]interface{}{
			"strings": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorJoinStrings(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockJoinStringsProcessSuccess() {
	// Given
	strings := []string{
		"string1",
		"string2",
	}
	block := blocks.NewBlockJoinStrings()
	data := &dataclasses.BlockData{
		Id:   "join_strings",
		Slug: "string-from-list-of-strings",
		Input: map[string]interface{}{
			"strings":   strings,
			"separator": "-_-",
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorJoinStrings(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Equal("string1-_-string2", result.String())
}

func (suite *UnitTestSuite) TestBlockJoinStringsProcessSuccessOneVideo() {
	// Given
	strings := []string{
		"string1",
	}
	block := blocks.NewBlockJoinStrings()
	data := &dataclasses.BlockData{
		Id:   "join_strings",
		Slug: "string-from-list-of-strings",
		Input: map[string]interface{}{
			"strings":   strings,
			"separator": "-_-",
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorJoinStrings(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Equal("string1", result.String())
}
