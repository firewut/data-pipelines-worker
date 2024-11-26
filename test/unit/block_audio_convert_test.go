package unit_test

import (
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockAudioConvert() {
	block := blocks.NewBlockAudioConvert()

	suite.Equal("audio_convert", block.GetId())
	suite.Equal("Audio Convert", block.GetName())
	suite.Equal("Convert Audio to to specified format audio", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("mp3", blockConfig.Format)
	suite.False(blockConfig.Mono)
	suite.Equal(44100, blockConfig.SampleRate)
	suite.Equal("64k", blockConfig.BitRate)
}

func (suite *UnitTestSuite) TestBlockAudioConvertValidateSchemaOk() {
	block := blocks.NewBlockAudioConvert()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockAudioConvertValidateSchemaFail() {
	block := blocks.NewBlockAudioConvert()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockAudioConvertProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockAudioConvert()
	data := &dataclasses.BlockData{
		Id:   "audio_convert",
		Slug: "audio-convert",
		Input: map[string]interface{}{
			"audio": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorAudioConvert(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockAudioConvertProcessSuccess() {
	// Given
	durationSeconds := 2

	audioBuffer := factories.GetShortAudioBufferMP3(durationSeconds)

	block := blocks.NewBlockAudioConvert()
	data := &dataclasses.BlockData{
		Id:   "audio_convert",
		Slug: "audio-convert",
		Input: map[string]interface{}{
			"audio": audioBuffer.Bytes(),
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorAudioConvert(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)
}
