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
	"mime/multipart"
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
	_config := config.GetConfig()
	_config.Storage.Local.RootPath = os.TempDir()

	// OpenAI
	openAIModelsList := `{
		"data": [
			{"id": "gpt-3.5-turbo"},
			{"id": "text-davinci-003"},
			{"id": "text-curie-001"}
		]
	}`
	modelsListEndpoint := suite.GetMockHTTPServerURL(openAIModelsList, http.StatusOK, 0)
	openaiClient := factories.NewOpenAIClient(modelsListEndpoint)
	_config.OpenAI.SetClient(openaiClient)

	// Telegram
	telegramBotInfo := `{
		"ok": true,
		"result": {
			"id": 123456789,
			"is_bot": true,
			"first_name": "Test Bot",
			"username": "test_bot"
		}
	}`
	telegramBotInfoEndpoint := suite.GetMockHTTPServerURL(telegramBotInfo, http.StatusOK, 0)
	telegramClient, err := factories.NewTelegramClient(telegramBotInfoEndpoint)
	suite.Nil(err)
	_config.Telegram.SetClient(telegramClient)

	suite.apiServers = make([]*APIServer, 0)
	for blockId, blockConfig := range suite._config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL(
				"Mocked Response OK",
				http.StatusOK,
				0,
			)
			suite._config.Blocks[blockId].Detector.Conditions["url"] = successUrl
		}
	}
}

func (suite *FunctionalTestSuite) GetMockHTTPServer(
	body string,
	statusCode int,
	responseDelay time.Duration,
	bodyMapping map[string][]string,
) *httptest.Server {
	suite.Lock()
	defer suite.Unlock()

	// Merge all body mappings into a single map
	var bodyMap map[string][]string
	bodyMapSliceIndex := make(map[string]int)

	if len(bodyMapping) > 0 {
		bodyMap = bodyMapping
	} else {
		bodyMap = make(map[string][]string)
	}

	serverMutex := &sync.Mutex{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)

		if responseDelay > 0 {
			time.Sleep(responseDelay)
		}

		serverMutex.Lock()
		defer serverMutex.Unlock()

		// Check if the requested path is in the bodyMap
		if responseBodySlice, exists := bodyMap[r.URL.Path]; exists {
			if _, exists := bodyMapSliceIndex[r.URL.Path]; !exists {
				bodyMapSliceIndex[r.URL.Path] = 0
			}

			w.Write([]byte(responseBodySlice[bodyMapSliceIndex[r.URL.Path]]))

			bodyMapSliceIndex[r.URL.Path] += 1
			if bodyMapSliceIndex[r.URL.Path] >= len(responseBodySlice) {
				bodyMapSliceIndex[r.URL.Path] = 0
			}
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
		make(map[string][]string),
	).URL
}

func (suite *FunctionalTestSuite) NewWorkerServerWithHandlers(
	available bool,
	_config config.Config,
) (*api.Server, interfaces.Worker, error) {
	suite.Lock()
	defer suite.Unlock()

	ctx, shutdown := context.WithCancel(context.Background())
	server, worker, err := factories.NewWorkerServerWithHandlers(
		ctx,
		available,
		_config,
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

func (suite *FunctionalTestSuite) SendProcessingStartRequestMultipart(
	server *api.Server,
	input schemas.PipelineStartInputSchema,
	httpClient *http.Client,
) (schemas.PipelineResumeOutputSchema, int, string, error) {
	var result schemas.PipelineResumeOutputSchema

	// Create a buffer for the multipart form data
	var requestBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&requestBody)

	err := multipartWriter.WriteField("pipeline.slug", input.Pipeline.Slug)
	suite.Nil(err)

	if input.Pipeline.ProcessingID != uuid.Nil {
		err = multipartWriter.WriteField("pipeline.processing_id", input.Pipeline.ProcessingID.String())
		suite.Nil(err)
	}

	err = multipartWriter.WriteField("block.slug", input.Block.Slug)
	suite.Nil(err)

	if input.Block.DestinationSlug != "" {
		_ = multipartWriter.WriteField("block.destination_slug", input.Block.DestinationSlug)
	}

	if input.Block.TargetIndex != -1 {
		err = multipartWriter.WriteField("block.target_index", fmt.Sprintf("%d", input.Block.TargetIndex))
		suite.Nil(err)
	}

	// Iterate through the Block.Input map to add each field and handle files
	for key, value := range input.Block.Input {
		switch v := value.(type) {
		case []byte: // File content or raw byte data
			ext, err := types.DetectMimeTypeFromBuffer(*bytes.NewBuffer(v))
			suite.Nil(err)

			fileField, err := multipartWriter.CreateFormFile(
				fmt.Sprintf("block.input.%s", key),
				fmt.Sprintf("%s.%s", key, ext),
			)
			suite.Nil(err)
			_, err = fileField.Write(v)
			suite.Nil(err)
		case string: // Regular form field
			err = multipartWriter.WriteField(fmt.Sprintf("block.input.%s", key), v)
			suite.Nil(err)
		case int, int32, int64, float32, float64: // Numbers as string
			err = multipartWriter.WriteField(fmt.Sprintf("block.input.%s", key), fmt.Sprintf("%v", v))
			suite.Nil(err)
		case bool: // Booleans as string
			err = multipartWriter.WriteField(fmt.Sprintf("block.input.%s", key), fmt.Sprintf("%t", v))
			suite.Nil(err)
		default:
		}
	}

	// Close the multipart writer to complete the form data
	err = multipartWriter.Close()
	suite.Nil(err)

	// Set up the HTTP client if not provided
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	// Send the HTTP request
	response, err := httpClient.Post(
		fmt.Sprintf("%s/pipelines/%s/start", server.GetAPIAddress(), input.Pipeline.Slug),
		multipartWriter.FormDataContentType(),
		&requestBody,
	)
	suite.Nil(err)
	defer response.Body.Close()

	responseBodyBytes, _ := io.ReadAll(response.Body)
	if err := json.Unmarshal(responseBodyBytes, &result); err != nil {
		return result, response.StatusCode, string(responseBodyBytes), err
	}

	return result, response.StatusCode, "", err
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

func (suite *FunctionalTestSuite) GetTelegramBotInfo() string {
	return `{
		"ok": true,
		"result": {
			"id": 123456789,
			"is_bot": true,
			"first_name": "Test Bot",
			"username": "test_bot"
		}
	}`
}
