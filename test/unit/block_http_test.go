package unit_test

import (
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/registries"
	"data-pipelines-worker/types/validators"
	"os"

	"net/http"
	"net/http/httptest"
)

func (suite *UnitTestSuite) TestBlockHTTP() {
	block := blocks.NewBlockHTTP()

	suite.Equal("http_request", block.GetId())
	suite.Equal("Request HTTP Resource", block.GetName())
	suite.Equal("Block to perform request to a URL and save the Response", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
}

func (suite *UnitTestSuite) TestBlockHTTPDetectOk() {
	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	blockDetected := block.Detect(
		&blocks.DetectorHTTP{
			Client: &http.Client{},
			Url:    server.URL,
		},
	)
	suite.True(blockDetected)
}

func (suite *UnitTestSuite) TestBlockHTTPDetectFail() {
	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	blockDetected := block.Detect(
		&blocks.DetectorHTTP{
			Client: &http.Client{},
			Url:    server.URL,
		},
	)
	suite.True(blockDetected)
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
	result, err := block.Process(&blocks.ProcessorHTTP{}, data)
	suite.Empty(result)
	suite.NotNil(err)
	suite.Contains(
		err.Error(),
		"Expected: string, given: null",
	)
}

func (suite *UnitTestSuite) TestBlockHTTPProcessSuccess() {
	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 200 OK
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, world!\n"))
		}),
	)
	defer server.Close()

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": server.URL,
		},
	}

	// Process the block
	result, err := block.Process(&blocks.ProcessorHTTP{}, data)
	suite.NotNil(result)
	suite.Nil(err)
	suite.Equal("Hello, world!\n", result.String())
}

func (suite *UnitTestSuite) TestBlockHTTPProcessError() {
	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 200 OK
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server panic"))
		}),
	)
	defer server.Close()

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": server.URL,
		},
	}

	// Process the block
	result, err := block.Process(&blocks.ProcessorHTTP{}, data)
	suite.NotNil(result)
	suite.NotNil(err)
	suite.Contains(result.String(), "Server panic")
	suite.Contains(
		err.Error(),
		"HTTP request failed with status code: 500",
	)
}

func (suite *UnitTestSuite) TestBlockHTTPSaveOutput() {
	registry, err := registries.NewPipelineRegistry(
		&dataclasses.PipelineCatalogueLoader{},
	)
	suite.Nil(err)

	pipelineSlug := "YT-CHANNEL-video-generation-block-prompt"
	pipeline := registry.Get(pipelineSlug)

	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 200 OK
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Testing save output method!\n"))
		}),
	)
	defer server.Close()

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": server.URL,
		},
	}
	data.SetPipeline(pipeline)
	data.SetBlock(block)

	// Process the block
	result, err := block.Process(&blocks.ProcessorHTTP{}, data)
	suite.NotNil(result)
	suite.Nil(err)
	suite.Equal("Testing save output method!\n", result.String())

	// Save the output
	fileName, err := block.SaveOutput(
		data,
		result,
		types.NewLocalStorage(),
	)
	suite.Nil(err)
	suite.NotEmpty(fileName)

	// Read file content
	content, err := os.ReadFile(fileName)
	suite.Nil(err)
	suite.Equal("Testing save output method!\n", string(content))
}

func (suite *UnitTestSuite) TestBlockHTTPSaveOutputNoSpaceOnDeviceLeft() {
	registry, err := registries.NewPipelineRegistry(
		&dataclasses.PipelineCatalogueLoader{},
	)
	suite.Nil(err)

	pipelineSlug := "YT-CHANNEL-video-generation-block-prompt"
	pipeline := registry.Get(pipelineSlug)

	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 200 OK
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Testing save output method!\n"))
		}),
	)
	defer server.Close()

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": server.URL,
		},
	}
	data.SetPipeline(pipeline)
	data.SetBlock(block)

	// Process the block
	result, err := block.Process(&blocks.ProcessorHTTP{}, data)
	suite.NotNil(result)
	suite.Nil(err)
	suite.Equal("Testing save output method!\n", result.String())

	// Save the output
	fileName, err := block.SaveOutput(
		data,
		result,
		&noSpaceLeftLocalStorage{},
	)
	suite.NotNil(err)
	suite.Empty(fileName)
}
