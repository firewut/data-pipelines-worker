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

func (suite *UnitTestSuite) TestBlockImageBlur() {
	block := blocks.NewBlockImageBlur()

	suite.Equal("image_blur", block.GetId())
	suite.Equal("Image Blur", block.GetName())
	suite.Equal("Blur Image", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
	suite.NotEmpty(block.GetConfigSection())

	blockConfig := &blocks.BlockImageBlurConfig{}
	helpers.MapToYAMLStruct(
		block.GetConfigSection(),
		blockConfig,
	)
	suite.Equal(1.5, blockConfig.Sigma)
}

func (suite *UnitTestSuite) TestBlockImageBlurValidateSchemaOk() {
	block := blocks.NewBlockImageBlur()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockImageBlurValidateSchemaFail() {
	block := blocks.NewBlockImageBlur()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockImageBlurProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockImageBlur()
	data := &dataclasses.BlockData{
		Id:   "image_blur",
		Slug: "image-blur",
		Input: map[string]interface{}{
			"image": nil,
		},
	}

	// When
	result, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorImageBlur(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockImageBlurProcessSuccess() {
	// Given
	width := 100
	height := 100

	sigmas := []float64{1.0, 2.0, 3.0, 4.0}

	type testCase struct {
		sigma float64
	}
	testCases := []testCase{}
	for _, s := range sigmas {
		testCases = append(testCases, testCase{s})
	}

	images := make([]image.Image, 0)
	for _, tc := range testCases {
		imageBuffer := suite.GetPNGImageBuffer(width, height)

		block := blocks.NewBlockImageBlur()
		data := &dataclasses.BlockData{
			Id:   "image_blur",
			Slug: "image-blur",
			Input: map[string]interface{}{
				"image": imageBuffer.Bytes(),
				"sigma": tc.sigma,
			},
		}

		// When
		result, err := block.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorImageBlur(),
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
