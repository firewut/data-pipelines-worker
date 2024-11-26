package unit_test

import (
	"net/http"

	"github.com/sashabaranov/go-openai"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockOpenAIRequestTranscription() {
	block := blocks.NewBlockOpenAIRequestTranscription()

	suite.Equal("openai_transcription_request", block.GetId())
	suite.Equal("OpenAI Audio Transcription", block.GetName())
	suite.Equal("Block to Transcribe audio to JSON using OpenAI. File size limit is 20MB and duration limit is 10 minutes.", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.EqualValues("whisper-1", blockConfig.Model)
	suite.EqualValues("en", blockConfig.Language)
	suite.EqualValues("verbose_json", blockConfig.Format)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTranscriptionValidateSchemaOk() {
	block := blocks.NewBlockOpenAIRequestTranscription()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTranscriptionValidateSchemaFail() {
	block := blocks.NewBlockOpenAIRequestTranscription()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTranscriptionProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockOpenAIRequestTranscription()
	data := &dataclasses.BlockData{
		Id:   "openai_transcription_request",
		Slug: "request-openai-transcription",
		Input: map[string]interface{}{
			"audio": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestTranscription(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTranscriptionProcessEmptyClient() {
	// Given
	block := blocks.NewBlockOpenAIRequestTranscription()
	data := &dataclasses.BlockData{
		Id:   "openai_transcription_request",
		Slug: "request-openai-transcription",
		Input: map[string]interface{}{
			"audio": "test.mp3",
		},
	}
	data.SetBlock(block)

	suite._config.OpenAI.SetClient(nil)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestTranscription(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTranscriptionProcessSuccess() {
	// Given
	block := blocks.NewBlockOpenAIRequestTranscription()
	data := &dataclasses.BlockData{
		Id:   "openai_transcription_request",
		Slug: "request-openai-transcription",
		Input: map[string]interface{}{
			"audio": []byte{1, 2, 3},
		},
	}
	data.SetBlock(block)

	mockedResponse := `{
		"duration": 18.719999313354492,
		"language": "english",
		"segments": [
			{
				"avg_logprob": -0.29384443163871765,
				"compression_ratio": 1.4321608543395996,
				"end": 6.28000020980835,
				"id": 0,
				"no_speech_prob": 0.001096802530810237,
				"seek": 0,
				"start": 0.0,
				"temperature": 0.0,
				"text": " On October 5, 2000, the United Nations established World Teachers' Day to celebrate educators",
				"tokens": [50364]
			}
		],
		"task": "transcribe",
		"text": "On October 5, 2000, the United Nations established World Teachers' Day to celebrate educators worldwide. This annual event honors the essential role teachers play in shaping futures, acknowledging their hard work and dedication in nurturing young minds and fostering lifelong learning."
	}`
	transcriptionAPIEndpoint := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)

	openaiClient := openai.NewClientWithConfig(
		openai.ClientConfig{
			BaseURL:            transcriptionAPIEndpoint,
			APIType:            openai.APITypeOpenAI,
			AssistantVersion:   "v2",
			OrgID:              "",
			HTTPClient:         &http.Client{},
			EmptyMessagesLimit: 0,
		},
	)

	suite._config.OpenAI.SetClient(openaiClient)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestTranscription(),
		data,
	)

	// Then
	suite.NotEmpty(result)
	suite.False(stop)
	suite.Nil(err)

	// Encode response to JSON
	suite.Contains(result[0].String(), `18.719999313354492`)
}
