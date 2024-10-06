package unit_test

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/sashabaranov/go-openai"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockOpenAIRequestImage() {
	block := blocks.NewBlockOpenAIRequestImage()

	suite.Equal("openai_image_request", block.GetId())
	suite.Equal("OpenAI Image Request", block.GetName())
	suite.Equal("Block to generate image from text using OpenAI Dall-e v3", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("standard", blockConfig.Quality)
	suite.Equal("1024x1024", blockConfig.Size)
	suite.Equal("natural", blockConfig.Style)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestImageValidateSchemaOk() {
	block := blocks.NewBlockOpenAIRequestImage()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestImageValidateSchemaFail() {
	block := blocks.NewBlockOpenAIRequestImage()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestImageProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockOpenAIRequestImage()
	data := &dataclasses.BlockData{
		Id:   "openai_image_request",
		Slug: "request-openai-image",
		Input: map[string]interface{}{
			"prompt": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestImage(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestImageProcessEmptyClient() {
	// Given
	block := blocks.NewBlockOpenAIRequestImage()
	data := &dataclasses.BlockData{
		Id:   "openai_image_request",
		Slug: "request-openai-image",
		Input: map[string]interface{}{
			"prompt": "Hello world!",
		},
	}
	data.SetBlock(block)

	suite._config.OpenAI.SetClient(nil)

	// When
	result, stop, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorOpenAIRequestImage(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockOpenAIRequestImageProcessSuccess() {
	// Given
	block := blocks.NewBlockOpenAIRequestImage()
	data := &dataclasses.BlockData{
		Id:   "openai_image_request",
		Slug: "request-openai-image",
		Input: map[string]interface{}{
			"prompt": "Hello world!",
		},
	}
	data.SetBlock(block)

	width, height := 256, 256
	pngBuffer := factories.GetPNGImageBuffer(width, height)
	mockedResponse := fmt.Sprintf(
		`{
			"created": 1683501845,
			"data": [
			  {
				"b64_json": "%s"
			  }
			]
		}`,
		base64.StdEncoding.EncodeToString(
			pngBuffer.Bytes(),
		),
	)
	imageAPIEndpoint := suite.GetMockHTTPServerURL(mockedResponse, http.StatusOK, 0)

	openaiClient := openai.NewClientWithConfig(
		openai.ClientConfig{
			BaseURL:            imageAPIEndpoint,
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
		blocks.NewProcessorOpenAIRequestImage(),
		data,
	)

	// Then
	suite.NotEmpty(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Equal(pngBuffer.Bytes(), result.Bytes())
}
