package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"github.com/sashabaranov/go-openai"
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
	_config.OpenAI.SetClient(openaiClient)

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

	_, shutdown := context.WithCancel(context.Background())
	server, worker, err := factories.NewWorkerServerWithHandlers(
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

	return server, worker, err
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

func (suite *FunctionalTestSuite) GetPNGImageBuffer(width int, height int) bytes.Buffer {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill it with white color
	_color := color.RGBA{100, 100, 100, 100}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, _color)
		}
	}

	// Draw lines every 50 pixels (vertical and horizontal)
	lineColor := color.RGBA{0, 0, 0, 255} // Black color for lines

	// Draw vertical lines
	for x := 0; x < width; x += 50 {
		for y := 0; y < height; y++ {
			img.Set(x, y, lineColor)
		}
	}

	// Draw horizontal lines
	for y := 0; y < height; y += 50 {
		for x := 0; x < width; x++ {
			img.Set(x, y, lineColor)
		}
	}

	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	suite.Nil(err)

	return *buf
}
