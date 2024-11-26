package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockFormatStringFromObject() {
	block := blocks.NewBlockFormatStringFromObject()

	suite.Equal("format_string_from_object", block.GetId())
	suite.Equal("Format String From Object", block.GetName())
	suite.Equal("Format string from object", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
}

func (suite *UnitTestSuite) TestBlockFormatStringFromObjectValidateSchemaOk() {
	block := blocks.NewBlockFormatStringFromObject()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockFormatStringFromObjectValidateSchemaFail() {
	block := blocks.NewBlockFormatStringFromObject()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockFormatStringFromObjectProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockFormatStringFromObject()
	data := &dataclasses.BlockData{
		Id:   "format_string_from_object",
		Slug: "text-format",
		Input: map[string]interface{}{
			"format": nil,
			"title":  nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorFormatStringFromObject(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockFormatStringFromObjectProcessSuccess() {
	// Given
	cases := []struct {
		template string
		title    string
		summary  string
		tags     []string
		expected string
	}{
		{
			template: "Title: {title}\nSummary: {summary}\nTags: {tags}",
			title:    "Big title",
			summary:  "Big summary",
			tags:     []string{"tag1", "tag2"},
			expected: "Title: Big title\nSummary: Big summary\nTags: tag1, tag2",
		},
		{
			template: "Title: {title}\nSummary: {summary}",
			title:    "Big title",
			summary:  "Big summary",
			tags:     nil,
			expected: "Title: Big title\nSummary: Big summary",
		},
		{
			template: "Title: {title}",
			title:    "Big title",
			summary:  "",
			tags:     nil,
			expected: "Title: Big title",
		},
		{
			template: "Title: {title}\nTags: [{tags}]",
			title:    "Big title",
			summary:  "",
			tags:     []string{"tag1", "tag2"},
			expected: "Title: Big title\nTags: [tag1, tag2]",
		},
		{
			template: "Title: {title}\nSummary: {summary}\nTags: {tags}",
			title:    "",
			summary:  "",
			tags:     nil,
			expected: "Title: \nSummary: \nTags: ",
		},
	}

	for _, tc := range cases {
		block := blocks.NewBlockFormatStringFromObject()
		data := &dataclasses.BlockData{
			Id:   "format_string_from_object",
			Slug: "format-string-from-object",
			Input: map[string]interface{}{
				"template": tc.template,
				"title":    tc.title,
				"summary":  tc.summary,
				"tags":     tc.tags,
			},
		}
		data.SetBlock(block)

		// When
		result, stop, _, _, _, err := block.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorFormatStringFromObject(),
			data,
		)

		// Then
		suite.NotNil(result)
		suite.False(stop)
		suite.Nil(err)

		suite.Equal(
			tc.expected,
			result[0].String(),
		)
	}
}
