package unit_test

import (
	"context"
	"net/http"
	"sync"
	"time"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
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
	suite.NotNil(err, err)
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
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorHTTP(),
		data,
	)
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
	suite.Contains(
		err.Error(),
		"Expected: string, given: null",
	)
}

func (suite *UnitTestSuite) TestBlockHTTPProcessSuccess() {
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!\n", http.StatusOK, 0)

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": successUrl,
		},
	}

	// Process the block
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorHTTP(),
		data,
	)

	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)
	suite.Equal("Hello, world!\n", result.String())
}

func (suite *UnitTestSuite) TestBlockHTTPProcessCancel() {
	// Given
	ctx, cancel := context.WithCancel(context.Background())

	successUrl := suite.GetMockHTTPServerURL("Hello, world!\n", http.StatusOK, time.Second)
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": successUrl,
		},
	}
	block := blocks.NewBlockHTTP()

	var wg sync.WaitGroup
	wg.Add(1)

	// Run the Process call in a separate goroutine
	var err error
	go func() {
		defer wg.Done()
		_, _, _, _, _, err = block.Process(
			ctx,
			blocks.NewProcessorHTTP(),
			data,
		)
	}()

	time.Sleep(time.Millisecond * 10)

	// When
	cancel()

	wg.Wait()

	// Then
	suite.NotNil(err, err)
	suite.Contains(err.Error(), "context canceled")
}

func (suite *UnitTestSuite) TestBlockHTTPProcessError() {
	block := blocks.NewBlockHTTP()

	failUrl := suite.GetMockHTTPServerURL("Server panic", http.StatusInternalServerError, 0)

	// Create a mock data
	data := &dataclasses.BlockData{
		Id:   "http_request",
		Slug: "http-request",
		Input: map[string]interface{}{
			"url": failUrl,
		},
	}

	// Process the block
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorHTTP(),
		data,
	)
	suite.NotNil(result)
	suite.False(stop)
	suite.NotNil(err, err)
	suite.Contains(result.String(), "Server panic")
	suite.Contains(
		err.Error(),
		"HTTP request failed with status code: 500",
	)
}
