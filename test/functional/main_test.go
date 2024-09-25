package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"data-pipelines-worker/api"
	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
)

type APIServer struct {
	Server   *api.Server
	Shutdown context.CancelFunc
}

type FunctionalTestSuite struct {
	sync.RWMutex

	suite.Suite
	_config config.Config

	httpTestServers []*httptest.Server // to mock http requests
	apiServers      []*APIServer
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
}

func (suite *FunctionalTestSuite) TearDownSuite() {
	suite.Lock()
	defer suite.Unlock()

}

func (suite *FunctionalTestSuite) SetupTest() {
	// Make Mock HTTP Server for each URL Block Detector
	_config := config.GetConfig()

	suite.apiServers = make([]*APIServer, 0)

	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL(
				"Mocked Response OK",
				http.StatusOK,
				0,
			)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl
		}
	}
}

func (suite *FunctionalTestSuite) GetMockHTTPServer(
	body string,
	statusCode int,
	responseDelay time.Duration,
	bodyMapping map[string]string,
) *httptest.Server {
	suite.Lock()
	defer suite.Unlock()

	// Merge all body mappings into a single map
	var bodyMap map[string]string
	if len(bodyMapping) > 0 {
		bodyMap = bodyMapping
	} else {
		bodyMap = make(map[string]string)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)

		if responseDelay > 0 {
			time.Sleep(responseDelay)
		}

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

func (suite *FunctionalTestSuite) GetMockHTTPServerURL(
	body string,
	statusCode int,
	responseDelay time.Duration,
) string {
	return suite.GetMockHTTPServer(
		body,
		statusCode,
		responseDelay,
		make(map[string]string),
	).URL
}

func (suite *FunctionalTestSuite) NewWorkerServerWithHandlers(available bool) (*api.Server, interfaces.Worker, error) {
	suite.Lock()
	defer suite.Unlock()

	ctx, shutdown := context.WithCancel(context.Background())
	server, _, err := factories.NewWorkerServerWithHandlers(
		context.Background(),
		available,
	)
	suite.Nil(err)
	suite.NotEmpty(server)

	server.GetPipelineRegistry().SetPipelineResultStorages(
		[]interfaces.Storage{
			types.NewLocalStorage(os.TempDir()),
		},
	)

	suite.apiServers = append(suite.apiServers, &APIServer{
		Server:   server,
		Shutdown: shutdown,
	})

	return factories.NewWorkerServerWithHandlers(ctx, true)
}

func (suite *FunctionalTestSuite) TearDownTest() {
	suite.Lock()
	defer suite.Unlock()

	for _, server := range suite.httpTestServers {
		server.Close()
	}
	suite.httpTestServers = make([]*httptest.Server, 0)

	for _, apiServer := range suite.apiServers {
		apiServer.Shutdown()
	}
	suite.apiServers = make([]*APIServer, 0)
}

func (suite *FunctionalTestSuite) GetTestPipeline(pipelineDefinition string) interfaces.Pipeline {
	pipeline, err := dataclasses.NewPipelineFromBytes([]byte(pipelineDefinition))
	suite.Nil(err)
	suite.NotEmpty(pipeline)

	return pipeline
}

func (suite *FunctionalTestSuite) GetTestPipelineTwoBlocks(firstBlockUrl string) interfaces.Pipeline {
	return suite.GetTestPipeline(
		fmt.Sprintf(`{
			"slug": "test-two-http-blocks",
			"title": "Test two HTTP Blocks",
			"description": "First Block makes a request to a URL and the second block makes a request result of First Block",
			"blocks": [
				{
					"id": "http_request",
					"slug": "test-block-first-slug",
					"description": "Request Local Resourse",
					"input": {
						"url": "%s"
					}
				},
				{
					"id": "http_request",
					"slug": "test-block-second-slug",
					"description": "Request Result from First Block",
					"input_config": {
						"property": {
							"url": {
								"origin": "test-block-first-slug"
							}
						}
					}
				}
			]
		}`, firstBlockUrl),
	)
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

func (suite *FunctionalTestSuite) SendProcessingStartRequest(
	server *api.Server,
	input schemas.PipelineStartInputSchema,
	httpClient *http.Client,
) (schemas.PipelineResumeOutputSchema, int, string, error) {
	var result schemas.PipelineResumeOutputSchema

	inputJSONData, err := json.Marshal(input)
	suite.Nil(err)

	if httpClient == nil {
		httpClient = &http.Client{}
	}

	response, err := httpClient.Post(
		fmt.Sprintf(
			"%s/pipelines/%s/start",
			server.GetAPIAddress(),
			input.Pipeline.Slug,
		),
		"application/json",
		bytes.NewReader(inputJSONData),
	)
	suite.Nil(err)
	defer response.Body.Close()

	responseBodyBytes, _ := io.ReadAll(response.Body)

	if err := json.Unmarshal(responseBodyBytes, &result); err != nil {
		return result, response.StatusCode, string(responseBodyBytes), err
	}

	return result, response.StatusCode, "", err
}
