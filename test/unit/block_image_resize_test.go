package unit_test

import (
	"bytes"
	"image"
	"image/png"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockImageResize() {
	block := blocks.NewBlockImageResize()

	suite.Equal("image_resize", block.GetId())
	suite.Equal("Image Resize", block.GetName())
	suite.Equal("Resize Image", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal(100, blockConfig.Width)
	suite.Equal(100, blockConfig.Height)
	suite.True(blockConfig.KeepAspectRatio)
}

func (suite *UnitTestSuite) TestBlockImageResizeValidateSchemaOk() {
	block := blocks.NewBlockImageResize()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockImageResizeValidateSchemaFail() {
	block := blocks.NewBlockImageResize()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockImageResizeProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockImageResize()
	data := &dataclasses.BlockData{
		Id:   "image_resize",
		Slug: "image-resize",
		Input: map[string]interface{}{
			"image": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorImageResize(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockImageResizeProcessSuccess() {
	// Given
	width := 100
	height := 100

	widths := []int{10, 20, 30, 40}
	heights := []int{10, 20, 30, 40}
	keepAspectRatios := []bool{false}

	type testCase struct {
		width           int
		height          int
		keepAspectRatio bool
	}
	testCases := []testCase{}
	for _, w := range widths {
		for _, h := range heights {
			for _, k := range keepAspectRatios {
				testCases = append(testCases, testCase{w, h, k})
			}
		}
	}

	images := make([]image.Image, 0)
	for _, tc := range testCases {
		imageBuffer := factories.GetPNGImageBuffer(width, height)

		block := blocks.NewBlockImageResize()
		data := &dataclasses.BlockData{
			Id:   "image_resize",
			Slug: "image-resize",
			Input: map[string]interface{}{
				"image":             imageBuffer.Bytes(),
				"width":             tc.width,
				"height":            tc.height,
				"keep_aspect_ratio": tc.keepAspectRatio,
			},
		}
		data.SetBlock(block)

		// When
		result, stop, _, _, _, err := block.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorImageResize(),
			data,
		)

		// Then
		suite.NotNil(result)
		suite.False(stop)
		suite.Nil(err)

		image, err := png.Decode(bytes.NewReader(result[0].Bytes()))
		suite.Nil(err)
		suite.NotNil(image)
		suite.Equal(tc.width, image.Bounds().Dx())
		suite.Equal(tc.height, image.Bounds().Dy())
		images = append(images, image)
	}

	// Check that all images are different
	for i := 0; i < len(images); i++ {
		for j := i + 1; j < len(images); j++ {
			suite.NotEqual(images[i], images[j])
		}
	}
}
