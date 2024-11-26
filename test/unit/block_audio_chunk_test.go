package unit_test

import (
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockAudioChunk() {
	block := blocks.NewBlockAudioChunk()

	suite.Equal("audio_chunk", block.GetId())
	suite.Equal("Audio Chunk", block.GetName())
	suite.Equal("Split Audio to chunks of specified duration", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("10m", blockConfig.Duration)
}

func (suite *UnitTestSuite) TestBlockAudioChunkValidateSchemaOk() {
	block := blocks.NewBlockAudioChunk()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockAudioChunkValidateSchemaFail() {
	block := blocks.NewBlockAudioChunk()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockAudioChunkProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockAudioChunk()
	data := &dataclasses.BlockData{
		Id:   "audio_chunk",
		Slug: "audio-chunk",
		Input: map[string]interface{}{
			"audio": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorAudioChunk(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockAudioChunkProcessSuccess() {
	// Given
	durationSeconds := 5

	audioBuffer := factories.GetShortAudioBufferMP3(durationSeconds)

	block := blocks.NewBlockAudioChunk()
	data := &dataclasses.BlockData{
		Id:   "audio_chunk",
		Slug: "audio-chunk",
		Input: map[string]interface{}{
			"audio":    audioBuffer.Bytes(),
			"duration": "1s",
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorAudioChunk(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.GreaterOrEqual(len(result), 5)
	for _, audioChunk := range result {
		suite.NotEmpty(audioChunk.Bytes())
	}
}
