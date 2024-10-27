package unit_test

import (
	"bytes"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockJoinVideos() {
	block := blocks.NewBlockJoinVideos()

	suite.Equal("join_videos", block.GetId())
	suite.Equal("Join Videos", block.GetName())
	suite.Equal("Join multiple videos into one", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal(false, blockConfig.ReEncode)
}

func (suite *UnitTestSuite) TestBlockJoinVideosValidateSchemaOk() {
	block := blocks.NewBlockJoinVideos()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockJoinVideosValidateSchemaFail() {
	block := blocks.NewBlockJoinVideos()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockJoinVideosProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockJoinVideos()
	data := &dataclasses.BlockData{
		Id:   "join_videos",
		Slug: "video-from-list-of-videos",
		Input: map[string]interface{}{
			"videos": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorJoinVideos(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockJoinVideosProcessSuccess() {
	// Given
	width := 10
	height := 10

	images := []bytes.Buffer{
		factories.GetPNGImageBuffer(width, height),
		factories.GetPNGImageBuffer(width, height),
	}
	videos := make([][]byte, len(images))
	for i, image := range images {
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
		_result, _stop, _, _, _, err := videoBlock.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorVideoFromImage(),
			_data,
		)

		suite.NotNil(_result)
		suite.False(_stop)
		suite.Nil(err)

		videos[i] = _result.Bytes()
	}

	block := blocks.NewBlockJoinVideos()
	data := &dataclasses.BlockData{
		Id:   "join_videos",
		Slug: "video-from-list-of-videos",
		Input: map[string]interface{}{
			"videos": videos,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorJoinVideos(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)
}
