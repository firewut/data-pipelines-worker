package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

const (
	textContent string = "Hello, this is a plain text file. It contains some text data."
	xmlContent  string = `<?xml version="1.0" encoding="UTF-8"?><note><to>T`
)

type UnitTestSuite struct {
	sync.RWMutex

	suite.Suite
	_config config.Config
	_blocks map[string]interfaces.Block

	// mock http requests
	httpTestServers []*httptest.Server

	// mock storages
	storages []interfaces.Storage

	// registries
	registries []generics.Registry[any]

	// contextCancels
	contextCancels []context.CancelFunc
}

func TestUnitTestSuite(t *testing.T) {
	// Set Logger level to debug
	config.GetLogger().SetLevel(log.INFO)

	suite.Run(t, new(UnitTestSuite))
}

func (suite *UnitTestSuite) SetupSuite() {
	suite.Lock()
	defer suite.Unlock()

	suite._config = config.GetConfig()
	suite._blocks = make(map[string]interfaces.Block)
	suite.httpTestServers = make([]*httptest.Server, 0)
	suite.storages = make([]interfaces.Storage, 0)
	suite.contextCancels = make([]context.CancelFunc, 0)
}

func (suite *UnitTestSuite) TearDownSuite() {
}

func (suite *UnitTestSuite) SetupTest() {
	// Make Mock HTTP Server for each URL Block Detector
	_config := config.GetConfig()
	_config.Storage.Local.RootPath = os.TempDir()

	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL("Mocked Response OK", http.StatusOK, 0)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl
		}
	}

	suite.registries = make([]generics.Registry[any], 0)
}

func (suite *UnitTestSuite) NewDummyBlock(id string) interfaces.Block {
	return &blocks.BlockParent{
		Id:           id,
		Name:         "Dummy Block",
		Description:  "This is a dummy block",
		Schema:       nil,
		SchemaPtr:    nil,
		SchemaString: "",
	}
}

func (suite *UnitTestSuite) GetMockHTTPServer(
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
		// Create a channel to listen for cancellation
		ctx := r.Context()

		select {
		case <-ctx.Done():
			// Context was canceled, exit early
			http.Error(w, "Request canceled", http.StatusRequestTimeout)
			return
		case <-time.After(responseDelay):
			// Introduce an artificial delay if specified
		}

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

func (suite *UnitTestSuite) GetMockHTTPServerURL(
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

func (suite *UnitTestSuite) TearDownTest() {
	suite.Lock()
	defer suite.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for _, server := range suite.httpTestServers {
		server.Close()
	}
	for _, storage := range suite.storages {
		storage.Shutdown()
	}
	for _, cancel := range suite.contextCancels {
		cancel()
	}
	for _, registry := range suite.registries {
		registry.Shutdown(ctx)
	}

	suite.contextCancels = make([]context.CancelFunc, 0)
	suite.httpTestServers = make([]*httptest.Server, 0)
	suite.storages = make([]interfaces.Storage, 0)
	suite.registries = make([]generics.Registry[any], 0)
}

func (suite *UnitTestSuite) GetTestPipelineDefinition() []byte {
	return []byte(`{
			"slug": "test-pipeline-slug",
			"title": "Test Pipeline",
			"description": "This is a test pipeline",
			"blocks": [
				{
					"id": "http_request",
					"slug": "get-text-content-for-video",
					"description": "Get Text Content of the Video",
					"input": {
						"url": "https://openai.com/api/v2/completion",
						"method": "POST",
						"headers": {
							"Content-Type": "application/json",
							"Authorisation: Bearer": "OPENAI_TOKEN"
						},
						"body": {
							"messages": [
								{"role": "assitant", "content": "theme"},
								{"role": "user", "content": "answer this question"}
							]
						}
					}
				}
			]
		}`,
	)
}

func (suite *UnitTestSuite) GetTestInputForProcessing(
	pipelineSlug string,
	blockSlug string,
	blockInput map[string]interface{},
) schemas.PipelineStartInputSchema {
	processingData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: pipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: blockSlug,
		},
	}
	if blockInput != nil {
		processingData.Block.Input = blockInput
	}

	return processingData
}

func (suite *UnitTestSuite) GetTestPipeline(pipelineDefinition string) interfaces.Pipeline {
	pipeline, err := dataclasses.NewPipelineFromBytes([]byte(pipelineDefinition))
	suite.Nil(err)
	suite.NotEmpty(pipeline)

	return pipeline
}

func (suite *UnitTestSuite) RegisterTestPipelineAndInputForProcessing(
	pipeline interfaces.Pipeline,
	pipelineSlug string,
	blockSlug string,
	blockInput map[string]interface{},
) (interfaces.Pipeline, schemas.PipelineStartInputSchema, *registries.PipelineRegistry) {
	processingData := suite.GetTestInputForProcessing(
		pipelineSlug,
		blockSlug,
		blockInput,
	)

	registry, err := registries.NewPipelineRegistry(
		suite.GetWorkerRegistry(),
		suite.GetBlockRegistry(),
		suite.GetProcessingRegistry(),
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	registry.Add(pipeline)

	return pipeline, processingData, registry
}

func (suite *UnitTestSuite) GetTestPipelineOneBlock(successUrl string) interfaces.Pipeline {
	return suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "test-pipeline-slug",
				"title": "Test Pipeline",
				"description": "Test Pipeline Description",
				"blocks": [
					{
						"id": "http_request",
						"slug": "test-block-slug",
						"description": "Request Local Resourse",
						"input": {
							"url": "%s"
						}
					}
				]
			}`,
			successUrl,
		),
	)
}

func (suite *UnitTestSuite) GetTestPipelineTwoBlocks(firstBlockUrl string) interfaces.Pipeline {
	return suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "test-pipeline-slug-two-blocks",
				"title": "Test Pipeline",
				"description": "Test Pipeline Description",
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
			}`,
			firstBlockUrl,
		),
	)
}

// Storage to simulate no space left on device
type noSpaceLeftLocalStorage struct{}

func (s *noSpaceLeftLocalStorage) GetStorageName() string {
	return "mock-no-space-left"
}
func (s *noSpaceLeftLocalStorage) GetStorageDirectory() string {
	return os.TempDir()
}

func (s *noSpaceLeftLocalStorage) NewStorageLocation(fileName string) interfaces.StorageLocation {
	return dataclasses.NewStorageLocation(s, s.GetStorageDirectory(), "", fileName)
}

func (s *noSpaceLeftLocalStorage) ListObjects(source interfaces.StorageLocation) ([]interfaces.StorageLocation, error) {
	return make([]interfaces.StorageLocation, 0), nil
}

func (s *noSpaceLeftLocalStorage) PutObject(source interfaces.StorageLocation, destination interfaces.StorageLocation) (interfaces.StorageLocation, error) {
	return s.NewStorageLocation(""), fmt.Errorf("No space left on device")
}

func (s *noSpaceLeftLocalStorage) PutObjectBytes(destination interfaces.StorageLocation, content *bytes.Buffer) (interfaces.StorageLocation, error) {
	return s.NewStorageLocation(""), fmt.Errorf("No space left on device")
}

func (s *noSpaceLeftLocalStorage) GetObject(source interfaces.StorageLocation, destination interfaces.StorageLocation) error {
	return fmt.Errorf("No space left on device")
}

func (s *noSpaceLeftLocalStorage) GetObjectBytes(source interfaces.StorageLocation) (*bytes.Buffer, error) {
	return nil, fmt.Errorf("No space left on device")
}

func (s *noSpaceLeftLocalStorage) DeleteObject(location interfaces.StorageLocation) error {
	return nil
}

func (s *noSpaceLeftLocalStorage) LocationExists(location interfaces.StorageLocation) bool {
	return false
}

func (s *noSpaceLeftLocalStorage) Shutdown() {}

type createdFile struct {
	filePath string
	data     *bytes.Buffer
}

type mockLocalStorage struct {
	sync.Mutex

	createdFilesChan chan createdFile
	storage          interfaces.Storage
	locations        []interfaces.StorageLocation
}

func (s *UnitTestSuite) NewMockLocalStorage(numBlocks int) *mockLocalStorage {
	mockLocalStorage := &mockLocalStorage{
		createdFilesChan: make(chan createdFile, numBlocks),
		storage: types.NewLocalStorage(
			os.TempDir(),
		),
		locations: make([]interfaces.StorageLocation, 0),
	}
	s.storages = append(s.storages, mockLocalStorage)

	return mockLocalStorage
}

func (s *mockLocalStorage) GetStorageName() string {
	return "mock-via-local"
}

func (s *mockLocalStorage) GetStorageDirectory() string {
	return s.storage.GetStorageDirectory()
}

func (s *mockLocalStorage) NewStorageLocation(fileName string) interfaces.StorageLocation {
	s.Lock()
	defer s.Unlock()

	storageLocation := s.storage.NewStorageLocation(fileName)
	s.locations = append(s.locations, storageLocation)

	return storageLocation
}

func (s *mockLocalStorage) ListObjects(source interfaces.StorageLocation) ([]interfaces.StorageLocation, error) {
	return s.storage.ListObjects(source)
}

func (s *mockLocalStorage) PutObject(source interfaces.StorageLocation, destination interfaces.StorageLocation) (interfaces.StorageLocation, error) {
	return s.storage.PutObject(source, destination)
}

func (s *mockLocalStorage) PutObjectBytes(destination interfaces.StorageLocation, content *bytes.Buffer) (interfaces.StorageLocation, error) {
	s.Lock()
	defer s.Unlock()

	s.createdFilesChan <- createdFile{
		filePath: destination.GetFilePath(),
		data:     bytes.NewBuffer(content.Bytes()),
	}

	return s.storage.PutObjectBytes(destination, content)
}

func (s *mockLocalStorage) GetObject(source interfaces.StorageLocation, destination interfaces.StorageLocation) error {
	return s.storage.GetObject(source, destination)
}

func (s *mockLocalStorage) GetObjectBytes(source interfaces.StorageLocation) (*bytes.Buffer, error) {
	return s.storage.GetObjectBytes(source)
}

func (s *mockLocalStorage) DeleteObject(location interfaces.StorageLocation) error {
	return s.storage.DeleteObject(location)
}

func (s *mockLocalStorage) LocationExists(location interfaces.StorageLocation) bool {
	return s.storage.LocationExists(location)
}

func (s *mockLocalStorage) AddFile(destination interfaces.StorageLocation, content *bytes.Buffer) (interfaces.StorageLocation, error) {
	return s.storage.PutObjectBytes(destination, content)
}

func (s *mockLocalStorage) GetCreatedFilesChan() chan createdFile {
	return s.createdFilesChan
}

func (s *mockLocalStorage) Shutdown() {
	s.Lock()
	defer s.Unlock()

	close(s.createdFilesChan)
	for _, location := range s.locations {
		s.storage.DeleteObject(location)
	}
}

func (s *UnitTestSuite) GetDiscoveredWorkers() []interfaces.Worker {
	discoveredEntries := []*zeroconf.ServiceEntry{
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "localhost",
			},
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.1")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
			Text:     []string{"version=0.1", "load=0.00", "available=false", "blocks="},
		},
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "remotehost",
			},
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.2")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
			Text:     []string{"version=0.1", "load=0.00", "available=true", "blocks=a,b,c"},
		},
	}
	discoveredWorkers := []interfaces.Worker{
		dataclasses.NewWorker(discoveredEntries[0]),
		dataclasses.NewWorker(discoveredEntries[1]),
	}

	return discoveredWorkers
}

func (suite *UnitTestSuite) GetMockServerHandlersResponse(
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

func (suite *UnitTestSuite) GetTestTranscriptionResult() string {
	return `{
		"text": "Segment one Content. Segment two Content",
		"task": "transcribe",
		"language": "english",
		"duration": 3,
		"segments": [
			{
				"id": 0,
				"seek": 0,
				"start": 0.0,
				"end": 1,
				"text": " Segment one Content.",
				"tokens": [50364],
				"temperature": 0.0,
				"avg_logprob": -0.22957643866539001,
				"compression_ratio": 1.3538461923599243,
				"no_speech_prob": 0.00005914297071285546
			},
			{
				"id": 1,
				"seek": 0,
				"start": 1,
				"end": 3,
				"text": " Segment two Content",
				"tokens": [50962],
				"temperature": 0.0,
				"avg_logprob": -0.22957643866539001,
				"compression_ratio": 1.3538461923599243,
				"no_speech_prob": 0.00005914297071285546
			}
		]
	}`
}

func (suite *UnitTestSuite) GetShutDownContext(
	duration time.Duration,
) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	suite.contextCancels = append(suite.contextCancels, cancel)

	return ctx
}

func (suite *UnitTestSuite) GetContextWithcancel() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	suite.contextCancels = append(suite.contextCancels, cancel)

	return ctx
}

func (suite *UnitTestSuite) GetWorkerRegistry(forceNewInstance ...bool) interfaces.WorkerRegistry {
	return registries.GetWorkerRegistry(forceNewInstance...)

}

func (suite *UnitTestSuite) GetBlockRegistry(forceNewInstance ...bool) interfaces.BlockRegistry {
	return registries.GetBlockRegistry(forceNewInstance...)

}

func (suite *UnitTestSuite) GetProcessingRegistry(forceNewInstance ...bool) interfaces.ProcessingRegistry {
	return registries.GetProcessingRegistry(forceNewInstance...)
}

func (suite *UnitTestSuite) GetPNGImageBuffer(width int, height int) bytes.Buffer {
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
