package unit_test

import (
	"bytes"
	"image"
	"image/png"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockImageAddText() {
	block := blocks.NewBlockImageAddText()

	suite.Equal("image_add_text", block.GetId())
	suite.Equal("Image Add Text", block.GetName())
	suite.Equal("Add text to Image", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
	suite.NotEmpty(block.GetConfigSection())

	blockConfig := &blocks.BlockImageAddTextConfig{}
	helpers.MapToYAMLStruct(
		block.GetConfigSection(),
		blockConfig,
	)
	suite.Equal(50.0, blockConfig.FontSize)
}

func (suite *UnitTestSuite) TestBlockImageAddTextValidateSchemaOk() {
	block := blocks.NewBlockImageAddText()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockImageAddTextValidateSchemaFail() {
	block := blocks.NewBlockImageAddText()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockImageAddTextProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockImageAddText()
	data := &dataclasses.BlockData{
		Id:   "image_add_text",
		Slug: "image-add-text",
		Input: map[string]interface{}{
			"text":  "test",
			"image": nil,
		},
	}

	// When
	result, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorImageAddText(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockImageAddTextProcessSuccess() {
	// Given
	width := 100
	height := 100

	texts := []string{"text1", "text2"}
	fontSizes := []int{20, 25}
	fontColors := []string{"#00FF00"}
	textPositions := []string{
		"top-left",
		"top-center",
		"top-right",
		"center-left",
		"center-center",
		"center-right",
		"bottom-left",
		"bottom-center",
		"bottom-right",
	}
	type testCase struct {
		text         string
		fontSize     int
		fontColor    string
		textPosition string
	}
	testCases := []testCase{}
	for _, text := range texts {
		for _, fontSize := range fontSizes {
			for _, fontColor := range fontColors {
				for _, textPosition := range textPositions {
					testCases = append(testCases, testCase{
						text:         text,
						fontSize:     fontSize,
						fontColor:    fontColor,
						textPosition: textPosition,
					})
				}
			}
		}
	}

	images := make([]image.Image, 0)
	for _, tc := range testCases {
		imageBuffer := suite.GetPNGImageBuffer(width, height)

		block := blocks.NewBlockImageAddText()
		data := &dataclasses.BlockData{
			Id:   "image_add_text",
			Slug: "image-add-text",
			Input: map[string]interface{}{
				"text":          tc.text,
				"font_size":     tc.fontSize,
				"font_color":    tc.fontColor,
				"text_position": tc.textPosition,
				"image":         imageBuffer.Bytes(),
			},
		}

		// When
		result, err := block.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorImageAddText(),
			data,
		)

		// Then
		suite.NotNil(result)
		suite.Nil(err)

		image, err := png.Decode(bytes.NewReader(result.Bytes()))
		suite.Nil(err)
		suite.NotNil(image)
		suite.Equal(width, image.Bounds().Dx())
		suite.Equal(height, image.Bounds().Dy())
		images = append(images, image)
	}

	// Check that all images are different
	for i := 0; i < len(images); i++ {
		for j := i + 1; j < len(images); j++ {
			suite.NotEqual(images[i], images[j])
		}
	}
}
