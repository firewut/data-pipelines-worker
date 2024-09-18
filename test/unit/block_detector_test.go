package unit_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/config"
)

func (suite *UnitTestSuite) TestDetectorHTTPSuccess() {
	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)

	detectorConfig := config.BlockConfigDetector{
		CheckInterval: time.Millisecond,
		Conditions: map[string]interface{}{
			"url": successUrl,
		},
	}

	detector := blocks.NewDetectorHTTP(&http.Client{}, detectorConfig)
	suite.True(detector.Detect())
}

func (suite *UnitTestSuite) TestDetectorHTTPFail() {
	server := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}),
	)
	defer server.Close()

	detectorConfig := config.BlockConfigDetector{
		CheckInterval: time.Millisecond,
		Conditions: map[string]interface{}{
			"url": server.URL,
		},
	}

	detector := blocks.NewDetectorHTTP(&http.Client{}, detectorConfig)
	suite.False(detector.Detect())
}

func (suite *UnitTestSuite) TestDetectorStartStop() {
	checkInterval := time.Millisecond * 1
	registryWg := &sync.WaitGroup{}
	block := suite.NewDummyBlock("test")
	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)

	detectorConfig := config.BlockConfigDetector{
		CheckInterval: checkInterval,
		Conditions: map[string]interface{}{
			"url": successUrl,
		},
	}

	detector := blocks.NewDetectorHTTP(&http.Client{}, detectorConfig)
	suite.False(block.IsAvailable())

	// Mock detection function
	detectFunction := func() bool {
		detector.Lock()
		defer detector.Unlock()

		return true
	}

	// Run detection Loop
	registryWg.Add(1)
	detector.Start(block, detectFunction)

	time.Sleep(checkInterval * 2)

	// Stop detection Loop
	detector.Stop(registryWg)
	registryWg.Wait()

	suite.True(block.IsAvailable())
}
