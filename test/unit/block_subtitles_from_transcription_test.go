package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockSubtitlesFromTranscription() {
	block := blocks.NewBlockSubtitlesFromTranscription()

	suite.Equal("subtitles_from_transcription", block.GetId())
	suite.Equal("Subtitles from Transcription", block.GetName())
	suite.Equal("Make Subtitles from Transcription", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("openai_verbose_json", blockConfig.InputFormat)
	suite.Equal("ass", blockConfig.OutputFormat)
	suite.Equal("Default", blockConfig.Name)
	suite.Equal("Arial", blockConfig.Fontname)
	suite.Equal(30, blockConfig.Fontsize)
	suite.Equal("&H00FFFFFF", blockConfig.PrimaryColour)
	suite.Equal("&H00000000", blockConfig.SecondaryColour)
	suite.Equal("&H00000000", blockConfig.BackColour)
	suite.Equal(-1, blockConfig.Bold)
	suite.Equal(0, blockConfig.Italic)
	suite.Equal(1, blockConfig.BorderStyle)
	suite.Equal(1.0, blockConfig.Outline)
	suite.Equal(0.0, blockConfig.Shadow)
	suite.Equal(2, blockConfig.Alignment)
	suite.Equal(10, blockConfig.MarginL)
	suite.Equal(10, blockConfig.MarginR)
	suite.Equal(10, blockConfig.MarginV)
}

func (suite *UnitTestSuite) TestBlockSubtitlesFromTranscriptionValidateSchemaOk() {
	block := blocks.NewBlockSubtitlesFromTranscription()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockSubtitlesFromTranscriptionValidateSchemaFail() {
	block := blocks.NewBlockSubtitlesFromTranscription()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockSubtitlesFromTranscriptionProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockSubtitlesFromTranscription()
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
		blocks.NewProcessorSubtitlesFromTranscription(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockSubtitlesFromTranscriptionProcessSuccess() {
	// Given
	openaiTranscription := `{"task":"transcribe","language":"english","duration":21.690000534057617,"segments":[{"id":0,"seek":0,"start":0,"end":8.140000343322754,"text":" On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.","tokens":[50364,1282,7617,1025,11,39498,11,264,1002,390,5680,3105,382,264,38376,4736,641,13828,2167,294,264,7051,13,50771],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":1,"seek":0,"start":8.140000343322754,"end":12.899999618530273,"text":" This marked the start of their legendary musical journey, leading to global fame.","tokens":[50771,639,12658,264,722,295,641,16698,9165,4671,11,5775,281,4338,16874,13,51009],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":2,"seek":0,"start":12.899999618530273,"end":16.739999771118164,"text":" Interestingly, John Lennon's harmonica playing added a distinct touch,","tokens":[51009,30564,11,2619,441,1857,266,311,14750,2262,2433,3869,257,10644,2557,11,51201],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":3,"seek":0,"start":16.739999771118164,"end":20.540000915527344,"text":" propelling them toward unprecedented stardom in the music industry.","tokens":[51201,25577,2669,552,7361,21555,342,515,298,294,264,1318,3518,13,51391],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false}],"words":null,"text":"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry."}`

	block := blocks.NewBlockSubtitlesFromTranscription()
	data := &dataclasses.BlockData{
		Id:   "subtitles_from_transcription",
		Slug: "subtitles-from-opena-transcription",
		Input: map[string]interface{}{
			"transcription": []byte(openaiTranscription),
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorSubtitlesFromTranscription(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(
		result.String(),
		`Dialogue: 0,00:00:16.73,00:00:20.54,Default,,0,0,0,, propelling them toward unprecedented `,
	)
}
