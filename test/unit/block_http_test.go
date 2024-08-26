package unit_test

import (
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/blocks"

	"net/http"
	"net/http/httptest"
)

func (suite *UnitTestSuite) TestBlockHTTP() {
	block := blocks.NewBlockHTTP()

	suite.Equal("http_request", block.GetId())
	suite.Equal("Request HTTP Resource", block.GetName())
	suite.Equal("Block to perform request to a URL and save the Response", block.GetDescription())
	suite.Nil(block.GetSchema())
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

	schemaPtr, schema, err := block.ValidateSchema(types.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockHTTPValidateSchemaFail() {
	block := blocks.NewBlockHTTP()

	block.SchemaString = "invalid schema"

	_, _, err := block.ValidateSchema(types.JSONSchemaValidator{})
	suite.NotNil(err)
}
