package unit_test

import (
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockVideoAddSubtitles() {
	block := blocks.NewBlockVideoAddSubtitles()

	suite.Equal("video_add_subtitles", block.GetId())
	suite.Equal("Video add Subtitles", block.GetName())
	suite.Equal("Add Subtitles to the Video. Mux or burn subtitles.", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("mux", blockConfig.EmbeddingType)
}

func (suite *UnitTestSuite) TestBlockVideoAddSubtitlesValidateSchemaOk() {
	block := blocks.NewBlockVideoAddSubtitles()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockVideoAddSubtitlesValidateSchemaFail() {
	block := blocks.NewBlockVideoAddSubtitles()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockVideoAddSubtitlesProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockVideoAddSubtitles()
	data := &dataclasses.BlockData{
		Id:   "video_add_subtitles",
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
		blocks.NewProcessorVideoAddSubtitles(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockVideoAddSubtitlesProcessSuccess() {
	// Given
	width := 10
	height := 10
	image := factories.GetPNGImageBuffer(width, height)
	openaiTranscription := `{"task":"transcribe","language":"english","duration":21.690000534057617,"segments":[{"id":0,"seek":0,"start":0,"end":8.140000343322754,"text":" On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.","tokens":[50364,1282,7617,1025,11,39498,11,264,1002,390,5680,3105,382,264,38376,4736,641,13828,2167,294,264,7051,13,50771],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":1,"seek":0,"start":8.140000343322754,"end":12.899999618530273,"text":" This marked the start of their legendary musical journey, leading to global fame.","tokens":[50771,639,12658,264,722,295,641,16698,9165,4671,11,5775,281,4338,16874,13,51009],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":2,"seek":0,"start":12.899999618530273,"end":16.739999771118164,"text":" Interestingly, John Lennon's harmonica playing added a distinct touch,","tokens":[51009,30564,11,2619,441,1857,266,311,14750,2262,2433,3869,257,10644,2557,11,51201],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":3,"seek":0,"start":16.739999771118164,"end":20.540000915527344,"text":" propelling them toward unprecedented stardom in the music industry.","tokens":[51201,25577,2669,552,7361,21555,342,515,298,294,264,1318,3518,13,51391],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false}],"words":null,"text":"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry."}`

	transcriptionBlock := blocks.NewBlockSubtitlesFromTranscription()
	transcriptionData := &dataclasses.BlockData{
		Id:   "subtitles_from_transcription",
		Slug: "subtitles-from-opena-transcription",
		Input: map[string]interface{}{
			"transcription": []byte(openaiTranscription),
		},
	}
	transcriptionData.SetBlock(transcriptionBlock)
	transcriptionResult, _, _, _, _, err := transcriptionBlock.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorSubtitlesFromTranscription(),
		transcriptionData,
	)
	suite.NotNil(transcriptionResult)
	suite.Nil(err)

	videoBlock := blocks.NewBlockVideoFromImage()
	videoData := &dataclasses.BlockData{
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
	videoData.SetBlock(videoBlock)
	videoResult, _, _, _, _, err := videoBlock.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorVideoFromImage(),
		videoData,
	)
	suite.NotNil(videoResult)
	suite.Nil(err)

	videoNoSubtitles := videoResult[0].Bytes()
	block := blocks.NewBlockVideoAddSubtitles()
	data := &dataclasses.BlockData{
		Id:   "video_add_subtitles",
		Slug: "video-add-audio",
		Input: map[string]interface{}{
			"video":     videoNoSubtitles,
			"subtitles": transcriptionResult[0].Bytes(),
		},
	}
	data.SetBlock(block)

	// When
	videoWithSubtitlesBuffer, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorVideoAddSubtitles(),
		data,
	)

	// Then
	suite.NotNil(videoWithSubtitlesBuffer)
	suite.False(stop)
	suite.Nil(err)

	suite.NotEqual(videoNoSubtitles, videoWithSubtitlesBuffer[0].Bytes())
}
