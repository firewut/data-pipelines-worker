package unit_test

import (
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockVideoAddAudio() {
	block := blocks.NewBlockVideoAddAudio()

	suite.Equal("video_add_audio", block.GetId())
	suite.Equal("Video add Audio", block.GetName())
	suite.Equal("Add Audio to the Video.", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal(false, blockConfig.ReplaceOriginalAudio)
}

func (suite *UnitTestSuite) TestBlockVideoAddAudioValidateSchemaOk() {
	block := blocks.NewBlockVideoAddAudio()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockVideoAddAudioValidateSchemaFail() {
	block := blocks.NewBlockVideoAddAudio()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockVideoAddAudioProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockVideoAddAudio()
	data := &dataclasses.BlockData{
		Id:   "video_add_audio",
		Slug: "video-add-audio",
		Input: map[string]interface{}{
			"video": nil,
			"audio": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorVideoAddAudio(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockVideoAddAudioProcessSuccess() {
	// Given
	width := 10
	height := 10

	image := factories.GetPNGImageBuffer(width, height)
	audioBuffer := factories.GetShortAudioBuffer(1)
	videoBlock := blocks.NewBlockVideoFromImage()
	_data := &dataclasses.BlockData{
		Id:   "video_from_image",
		Slug: "video-from-image",
		Input: map[string]interface{}{
			"image":  image.Bytes(),
			"start":  0.0,
			"end":    1.0,
			"fps":    1,
			"preset": "veryfast",
			"crf":    23,
		},
	}
	_data.SetBlock(videoBlock)
	_videoResult, _stop, _, _, _, err := videoBlock.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorVideoFromImage(),
		_data,
	)

	suite.NotNil(_videoResult)
	suite.False(_stop)
	suite.Nil(err)

	videoNoAudio := _videoResult.Bytes()

	block := blocks.NewBlockVideoAddAudio()
	data := &dataclasses.BlockData{
		Id:   "video_add_audio",
		Slug: "video-add-audio",
		Input: map[string]interface{}{
			"video": videoNoAudio,
			"audio": audioBuffer.Bytes(),
		},
	}
	data.SetBlock(block)

	// When
	videoWithAudioBuffer, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorVideoAddAudio(),
		data,
	)

	// Then
	suite.NotNil(videoWithAudioBuffer)
	suite.False(stop)
	suite.Nil(err)

	suite.NotEqual(videoNoAudio, videoWithAudioBuffer.Bytes())
}
