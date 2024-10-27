package unit_test

import (
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockVideoFromImage() {
	block := blocks.NewBlockVideoFromImage()

	suite.Equal("video_from_image", block.GetId())
	suite.Equal("Video from image", block.GetName())
	suite.Equal("Make video from image", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal(30, blockConfig.FPS)
	suite.Equal("mp4", blockConfig.Format)
	suite.Equal(0.0, blockConfig.Start)
	suite.Equal(1.0, blockConfig.End)
	suite.Equal("veryfast", blockConfig.Preset)
	suite.Equal(23, blockConfig.CRF)
}

func (suite *UnitTestSuite) TestBlockVideoFromImageValidateSchemaOk() {
	block := blocks.NewBlockVideoFromImage()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockVideoFromImageValidateSchemaFail() {
	block := blocks.NewBlockVideoFromImage()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockVideoFromImageProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockVideoFromImage()
	data := &dataclasses.BlockData{
		Id:   "video_from_image",
		Slug: "video-from-image",
		Input: map[string]interface{}{
			"image": nil,
			"start": 1.0,
			"end":   2.0,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorVideoFromImage(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockVideoFromImageProcessSuccess() {
	// Given
	width := 10
	height := 10

	imageBuffer := factories.GetPNGImageBuffer(width, height)

	block := blocks.NewBlockVideoFromImage()
	data := &dataclasses.BlockData{
		Id:   "video_from_image",
		Slug: "video-from-image",
		Input: map[string]interface{}{
			"image":  imageBuffer.Bytes(),
			"start":  0.0,
			"end":    1.0,
			"fps":    1,
			"preset": "veryfast",
			"crf":    23,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorVideoFromImage(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)
}
