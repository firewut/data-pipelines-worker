package unit_test

import (
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockAudioFromVideo() {
	block := blocks.NewBlockAudioFromVideo()

	suite.Equal("audio_from_video", block.GetId())
	suite.Equal("Audio from Video", block.GetName())
	suite.Equal("Extact audio from video", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("mp3", blockConfig.Format)
	suite.Equal(-1.0, blockConfig.Start)
	suite.Equal(-1.0, blockConfig.End)
}

func (suite *UnitTestSuite) TestBlockAudioFromVideoValidateSchemaOk() {
	block := blocks.NewBlockAudioFromVideo()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockAudioFromVideoValidateSchemaFail() {
	block := blocks.NewBlockAudioFromVideo()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockAudioFromVideoProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockAudioFromVideo()
	data := &dataclasses.BlockData{
		Id:   "audio_from_video",
		Slug: "audio-from-video",
		Input: map[string]interface{}{
			"video": nil,
			"start": 1.0,
			"end":   2.0,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorAudioFromVideo(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockAudioFromVideoProcessSuccess() {
	// Given
	durationSeconds := 2
	width := 100
	height := 100

	videoBuffer := factories.GetShortVideoBuffer(width, height, durationSeconds)

	block := blocks.NewBlockAudioFromVideo()
	data := &dataclasses.BlockData{
		Id:   "audio_from_video",
		Slug: "audio-from-video",
		Input: map[string]interface{}{
			"video": videoBuffer.Bytes(),
			"start": 0.0,
			"end":   1.5,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorAudioFromVideo(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)
}
