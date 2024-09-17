package functional_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"data-pipelines-worker/api"
	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
)

type FunctionalTestSuite struct {
	sync.RWMutex

	suite.Suite
	server  *api.Server
	_config config.Config

	httpTestServers []*httptest.Server // to mock http requests
}

func TestFunctionalTestSuite(t *testing.T) {
	// Set Logger level to debug
	config.GetLogger().SetLevel(log.INFO)

	suite.Run(t, new(FunctionalTestSuite))
}

func (suite *FunctionalTestSuite) SetupSuite() {
	suite.Lock()
	defer suite.Unlock()

	suite._config = config.GetConfig()
	suite.server = factories.ServerFactory()
}

func (suite *FunctionalTestSuite) GetServer() *api.Server {
	return suite.server
}

func (suite *FunctionalTestSuite) TearDownSuite() {
	suite.Lock()
	defer suite.Unlock()

	suite.server.Shutdown(time.Second)
}

func (suite *FunctionalTestSuite) SetupTest() {
	// Make Mock HTTP Server for each URL Block Detector
	_config := config.GetConfig()
	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL("Mocked Response OK", http.StatusOK)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl
		}
	}
}

func (suite *FunctionalTestSuite) GetMockHTTPServer(
	body string,
	statusCode int,
	bodyMapping ...map[string]string,
) *httptest.Server {
	suite.Lock()
	defer suite.Unlock()

	// Merge all body mappings into a single map
	var bodyMap map[string]string
	if len(bodyMapping) > 0 {
		bodyMap = bodyMapping[0]
	} else {
		bodyMap = make(map[string]string)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)

		// Check if the requested path is in the bodyMap
		if responseBody, exists := bodyMap[r.URL.Path]; exists {
			w.Write([]byte(responseBody))
		} else if body != "" {
			// Default body if no specific path is matched
			w.Write([]byte(body))
		}
	}))
	suite.httpTestServers = append(suite.httpTestServers, server)

	return server
}

func (suite *FunctionalTestSuite) GetMockHTTPServerURL(body string, statusCode int) string {
	return suite.GetMockHTTPServer(body, statusCode).URL
}

func (suite *FunctionalTestSuite) TearDownTest() {
	suite.Lock()
	defer suite.Unlock()

	for _, server := range suite.httpTestServers {
		server.Close()
	}
	suite.httpTestServers = make([]*httptest.Server, 0)
}

func (suite *FunctionalTestSuite) GetTestPipeline(pipelineDefinition string) interfaces.Pipeline {
	pipeline, err := dataclasses.NewPipelineFromBytes([]byte(pipelineDefinition))
	suite.Nil(err)
	suite.NotEmpty(pipeline)

	return pipeline
}

func (suite *FunctionalTestSuite) GetMockServerHandlersResponse(
	pipelines map[string]interfaces.Pipeline,
	blocks map[string]interfaces.Block,
	startProcessingId uuid.UUID,
	resumeProcessingId uuid.UUID,
) map[string]string {
	pipelinesResponse, err := json.Marshal(pipelines)
	suite.Nil(err)

	blocksResponse, err := json.Marshal(blocks)
	suite.Nil(err)

	mockedResponses := map[string]string{}

	for _, pipeline := range pipelines {
		startProcessingIdResponse, err := json.Marshal(
			schemas.PipelineResumeOutputSchema{
				ProcessingID: startProcessingId,
			},
		)
		suite.Nil(err)
		resumeProcessingIdResponse, err := json.Marshal(
			schemas.PipelineResumeOutputSchema{
				ProcessingID: resumeProcessingId,
			},
		)
		suite.Nil(err)

		mockedResponses[fmt.Sprintf("/pipelines/%s/start", pipeline.GetSlug())] = string(startProcessingIdResponse)
		mockedResponses[fmt.Sprintf("/pipelines/%s/resume", pipeline.GetSlug())] = string(resumeProcessingIdResponse)
	}

	mockedResponses["/pipelines"] = string(pipelinesResponse)
	mockedResponses["/blocks"] = string(blocksResponse)

	return mockedResponses
}
