package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

func (suite *UnitTestSuite) TestBlockOpenAIRequestCompletion() {
	block := blocks.NewBlockOpenAIRequestCompletion()

	suite.Equal("openai_chat_completion", block.GetId())
	suite.Equal("OpenAI Chat Completion", block.GetName())
	suite.Equal("Block to perform request to OpenAI's Chat Completion", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.NotEmpty(blockConfig)
	suite.Equal("gpt-4o-2024-08-06", blockConfig.Model)
	suite.Equal("You are a helpful assistant.", blockConfig.SystemPrompt)
	suite.Equal("Hello ChatGPT, how are you?", blockConfig.UserPrompt)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestCompletionValidateSchemaOk() {
	block := blocks.NewBlockOpenAIRequestCompletion()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestCompletionValidateSchemaFail() {
	block := blocks.NewBlockOpenAIRequestCompletion()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestCompletionProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockOpenAIRequestCompletion()
	data := &dataclasses.BlockData{
		Id:   "openai_chat_completion",
		Slug: "request-openai-chat-completion",
		Input: map[string]interface{}{
			"model":       nil,
			"user_prompt": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestCompletion(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestCompletionProcessEmptyClient() {
	// Given
	block := blocks.NewBlockOpenAIRequestCompletion()
	data := &dataclasses.BlockData{
		Id:   "openai_chat_completion",
		Slug: "request-openai-chat-completion",
		Input: map[string]interface{}{
			"user_prompt": "Hello world!",
		},
	}
	data.SetBlock(block)

	suite._config.OpenAI.SetClient(nil)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestCompletion(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestCompletionProcessSuccess() {
	// Given
	block := blocks.NewBlockOpenAIRequestCompletion()
	data := &dataclasses.BlockData{
		Id:   "openai_chat_completion",
		Slug: "request-openai-chat-completion",
		Input: map[string]interface{}{
			"system_prompt": "You are a helpful assistant.",
			"user_prompt":   "Hello world!",
		},
	}
	data.SetBlock(block)

	mockedResponse := `{
		"id":"chatcmpl-123",
		"object":"chat.completion",
		"created":1677652288,
		"model":"gpt-4o-2024-08-06",
		"system_fingerprint":"fp_44709d6fcb",
		"choices":[
			{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"\n\nHello there, how may I assist you today?"
				},
				"logprobs":null,
				"finish_reason":"stop"
			}
		],
		"usage":{
			"prompt_tokens":9,
			"completion_tokens":12,
			"total_tokens":21,
			"completion_tokens_details":{
				"reasoning_tokens":0
			}
		}
	}`
	modelsListEndpoint := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)

	openaiClient := openai.NewClientWithConfig(
		openai.ClientConfig{
			BaseURL:            modelsListEndpoint,
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
		blocks.NewProcessorOpenAIRequestCompletion(),
		data,
	)

	// Then
	suite.NotEmpty(result)
	suite.False(stop)
	suite.Nil(err)
}
