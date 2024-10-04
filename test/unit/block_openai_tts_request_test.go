package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

func (suite *UnitTestSuite) TestBlockOpenAIRequestTTS() {
	block := blocks.NewBlockOpenAIRequestTTS()

	suite.Equal("openai_tts_request", block.GetId())
	suite.Equal("OpenAI Text to Speech", block.GetName())
	suite.Equal("Block to generate audio from text using OpenAI", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.EqualValues("tts-1", blockConfig.Model)
	suite.Equal("Hello ChatGPT, how are you?", blockConfig.Text)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTTSValidateSchemaOk() {
	block := blocks.NewBlockOpenAIRequestTTS()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTTSValidateSchemaFail() {
	block := blocks.NewBlockOpenAIRequestTTS()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTTSProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockOpenAIRequestTTS()
	data := &dataclasses.BlockData{
		Id:   "openai_tts_request",
		Slug: "request-openai-tts",
		Input: map[string]interface{}{
			"text": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestTTS(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTTSProcessEmptyClient() {
	// Given
	block := blocks.NewBlockOpenAIRequestTTS()
	data := &dataclasses.BlockData{
		Id:   "openai_tts_request",
		Slug: "request-openai-tts",
		Input: map[string]interface{}{
			"text": "Hello world!",
		},
	}
	data.SetBlock(block)

	suite._config.OpenAI.SetClient(nil)

	// When
	result, stop, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestTTS(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestTTSProcessSuccess() {
	// Given
	block := blocks.NewBlockOpenAIRequestTTS()
	data := &dataclasses.BlockData{
		Id:   "openai_tts_request",
		Slug: "request-openai-tts",
		Input: map[string]interface{}{
			"text": "Hello world!",
		},
	}
	data.SetBlock(block)

	mockedResponse := `tts-content`
	ttsAPIEndpoint := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)

	openaiClient := openai.NewClientWithConfig(
		openai.ClientConfig{
			BaseURL:            ttsAPIEndpoint,
			APIType:            openai.APITypeOpenAI,
			AssistantVersion:   "v2",
			OrgID:              "",
			HTTPClient:         &http.Client{},
			EmptyMessagesLimit: 0,
		},
	)

	suite._config.OpenAI.SetClient(openaiClient)

	// When
	result, stop, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestTTS(),
		data,
	)

	// Then
	suite.NotEmpty(result)
	suite.False(stop)
	suite.Nil(err)
}
