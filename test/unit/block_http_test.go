package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"net/http"
	"net/http/httptest"
)

func (suite *UnitTestSuite) TestBlockHTTP() {
	block := blocks.NewBlockHTTP()

	suite.Equal("get_http_url", block.GetId())
	suite.Equal("Get HTTP URL", block.GetName())
	suite.Equal("Block to perform request to a URL and get the content", block.GetDescription())
	suite.NotEmpty(block.GetSchema())
}

func (suite *UnitTestSuite) TestBlockHTTPDetectOk() {
	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpClient := &http.Client{}

	blockDetected := block.Detect(httpClient, server.URL)
	suite.True(blockDetected)
}

func (suite *UnitTestSuite) TestBlockHTTPDetectFail() {
	block := blocks.NewBlockHTTP()

	// Create a mock server that always returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	httpClient := &http.Client{}

	blockDetected := block.Detect(httpClient, server.URL)
	suite.True(blockDetected)
}
