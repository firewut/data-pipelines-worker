package unit_test

import (
	"os"

	"net/http"

	"github.com/google/uuid"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/registries"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockHTTP() {
	block := blocks.NewBlockHTTP()

	suite.Equal("http_request", block.GetId())
	suite.Equal("Request HTTP Resource", block.GetName())
	suite.Equal("Block to perform request to a URL and save the Response", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
}

func (suite *UnitTestSuite) TestBlockHTTPValidateSchemaOk() {
	block := blocks.NewBlockHTTP()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockHTTPValidateSchemaFail() {
	block := blocks.NewBlockHTTP()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockHTTPProcessIncorrectInput() {
	block := blocks.NewBlockHTTP()

	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": nil,
		},
	}

	// Process the block
	result, err := block.Process(blocks.NewProcessorHTTP(), data)
	suite.Empty(result)
	suite.NotNil(err)
	suite.Contains(
		err.Error(),
		"Expected: string, given: null",
	)
}

func (suite *UnitTestSuite) TestBlockHTTPProcessSuccess() {
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!\n", http.StatusOK)

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": successUrl,
		},
	}

	// Process the block
	result, err := block.Process(blocks.NewProcessorHTTP(), data)
	suite.NotNil(result)
	suite.Nil(err)
	suite.Equal("Hello, world!\n", result.String())
}

func (suite *UnitTestSuite) TestBlockHTTPProcessError() {
	block := blocks.NewBlockHTTP()

	failUrl := suite.GetMockHTTPServerURL("Server panic", http.StatusInternalServerError)

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": failUrl,
		},
	}

	// Process the block
	result, err := block.Process(blocks.NewProcessorHTTP(), data)
	suite.NotNil(result)
	suite.NotNil(err)
	suite.Contains(result.String(), "Server panic")
	suite.Contains(
		err.Error(),
		"HTTP request failed with status code: 500",
	)
}

func (suite *UnitTestSuite) TestBlockHTTPSaveOutputLocalStorage() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipelineSlug := "YT-CHANNEL-video-generation-block-prompt"
	pipeline := registry.Get(pipelineSlug)
	processingId := uuid.New()

	block := blocks.NewBlockHTTP()

	httpContent := "Testing save output method!\n"
	successUrl := suite.GetMockHTTPServerURL(httpContent, http.StatusOK)

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": successUrl,
		},
	}
	data.SetPipeline(pipeline)
	data.SetBlock(block)

	// Process the block
	result, err := block.Process(blocks.NewProcessorHTTP(), data)
	suite.NotNil(result)
	suite.Nil(err)
	suite.Equal(httpContent, result.String())

	// Save the output
	fileName, err := block.SaveOutput(
		pipeline.GetSlug(),
		data.GetSlug(),
		result,
		0,
		processingId,
		types.NewLocalStorage(""),
	)
	suite.Nil(err)
	suite.NotEmpty(fileName)
	defer os.Remove(fileName)

	// Read file content
	content, err := os.ReadFile(fileName)
	suite.Nil(err)
	suite.Equal(httpContent, string(content))
}

func (suite *UnitTestSuite) TestBlockHTTPSaveOutputNoSpaceOnDeviceLeft() {
	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	pipelineSlug := "YT-CHANNEL-video-generation-block-prompt"
	pipeline := registry.Get(pipelineSlug)

	block := blocks.NewBlockHTTP()

	httpContent := "Testing save output method!\n"
	successUrl := suite.GetMockHTTPServerURL(httpContent, http.StatusOK)

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": successUrl,
		},
	}
	data.SetPipeline(pipeline)
	data.SetBlock(block)

	// Process the block
	result, err := block.Process(blocks.NewProcessorHTTP(), data)
	suite.NotNil(result)
	suite.Nil(err)
	suite.Equal(httpContent, result.String())

	// Save the output
	fileName, err := block.SaveOutput(
		pipeline.GetSlug(),
		data.GetSlug(),
		result,
		0,
		uuid.New(),
		&noSpaceLeftLocalStorage{},
	)
	suite.NotNil(err)
	suite.Empty(fileName)
}
