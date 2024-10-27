package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockTextAddPrefixOrSuffix() {
	block := blocks.NewBlockTextAddPrefixOrSuffix()

	suite.Equal("wrap_text", block.GetId())
	suite.Equal("Text Add Prefix or Suffix", block.GetName())
	suite.Equal("Add a prefix and/or suffix to a text", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
}

func (suite *UnitTestSuite) TestBlockTextAddPrefixOrSuffixValidateSchemaOk() {
	block := blocks.NewBlockTextAddPrefixOrSuffix()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockTextAddPrefixOrSuffixValidateSchemaFail() {
	block := blocks.NewBlockTextAddPrefixOrSuffix()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockTextAddPrefixOrSuffixProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockTextAddPrefixOrSuffix()
	data := &dataclasses.BlockData{
		Id:   "wrap_text",
		Slug: "text-replace",
		Input: map[string]interface{}{
			"text":   nil,
			"prefix": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorTextAddPrefixOrSuffix(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockTextAddPrefixOrSuffixProcessSuccess() {
	// Given
	cases := []struct {
		text   string
		prefix string
		suffix string
	}{
		{
			text:   "Big text",
			prefix: "Prefix",
			suffix: "Suffix",
		},
		{
			text:   "Big text",
			prefix: "Prefix",
			suffix: "",
		},
		{
			text:   "Big text",
			prefix: "",
			suffix: "Suffix",
		},
		{
			text:   "Big text",
			prefix: "",
			suffix: "",
		},
	}

	for _, tc := range cases {
		block := blocks.NewBlockTextAddPrefixOrSuffix()
		data := &dataclasses.BlockData{
			Id:   "wrap_text",
			Slug: "text-add-prefix-or-suffix",
			Input: map[string]interface{}{
				"text":   tc.text,
				"prefix": tc.prefix,
				"suffix": tc.suffix,
			},
		}
		data.SetBlock(block)

		// When
		result, stop, _, _, _, err := block.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorTextAddPrefixOrSuffix(),
			data,
		)

		// Then
		suite.NotNil(result)
		suite.False(stop)
		suite.Nil(err)

		suite.Equal(
			tc.prefix+tc.text+tc.suffix,
			result.String(),
		)
	}
}
