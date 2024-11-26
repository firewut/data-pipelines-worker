package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockTextReplace() {
	block := blocks.NewBlockTextReplace()

	suite.Equal("text_replace", block.GetId())
	suite.Equal("Text Replace", block.GetName())
	suite.Equal("Replace text in a string", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
}

func (suite *UnitTestSuite) TestBlockTextReplaceValidateSchemaOk() {
	block := blocks.NewBlockTextReplace()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockTextReplaceValidateSchemaFail() {
	block := blocks.NewBlockTextReplace()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockTextReplaceProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockTextReplace()
	data := &dataclasses.BlockData{
		Id:   "text_replace",
		Slug: "text-replace",
		Input: map[string]interface{}{
			"text": nil,
			"old":  nil,
			"new":  nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorTextReplace(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockTextReplaceProcessSuccess() {
	// Given
	text := "On October 14, 1066, the Battle of Hastings took place, a pivotal " +
		"moment in English history. William the Conqueror's Norman forces defeated " +
		"King Harold II's army, leading to Norman rule in England. " +
		"This marked the beginning of significant cultural and political changes."
	_old := `William the Conqueror's Norman forces defeated King Harold II's army, leading to Norman`
	_new := _old
	_prefix := "<PREFIX>"
	_suffix := "<SUFFIX>"

	block := blocks.NewBlockTextReplace()
	data := &dataclasses.BlockData{
		Id:   "text_replace",
		Slug: "text-replace",
		Input: map[string]interface{}{
			"text":   text,
			"old":    _old,
			"new":    _new,
			"prefix": _prefix,
			"suffix": _suffix,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorTextReplace(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Equal(
		"On October 14, 1066, the Battle of Hastings took place, a pivotal "+
			"moment in English history. <PREFIX>William the Conqueror's Norman forces defeated "+
			"King Harold II's army, leading to Norman<SUFFIX> rule in England. "+
			"This marked the beginning of significant cultural and political changes.",
		result[0].String(),
	)
}
