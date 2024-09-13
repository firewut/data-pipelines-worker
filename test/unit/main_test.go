package unit_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"

	"github.com/grandcat/zeroconf"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"
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

	httpTestServers []*httptest.Server // to mock http requests
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
}

func (suite *UnitTestSuite) TearDownSuite() {
}

func (suite *UnitTestSuite) SetupTest() {
	// Make Mock HTTP Server for each URL Block Detector
	_config := config.GetConfig()
	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL("Mocked Response OK", http.StatusOK)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl
		}
	}
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

func (suite *UnitTestSuite) GetMockHTTPServerURL(body string, statusCode int) string {
	return suite.GetMockHTTPServer(body, statusCode).URL
}

func (suite *UnitTestSuite) TearDownTest() {
	suite.Lock()
	defer suite.Unlock()

	for _, server := range suite.httpTestServers {
		server.Close()
	}
	suite.httpTestServers = make([]*httptest.Server, 0)
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

func (s *noSpaceLeftLocalStorage) ListObjects(bucket string) ([]string, error) {
	return make([]string, 0), nil
}

func (s *noSpaceLeftLocalStorage) PutObject(bucket, filePath string) error {
	return fmt.Errorf("No space left on device")
}

func (s *noSpaceLeftLocalStorage) PutObjectBytes(directory string, buffer *bytes.Buffer) (string, error) {
	return "", fmt.Errorf("No space left on device")
}

func (s *noSpaceLeftLocalStorage) GetObject(bucket, objectName string, filePath string) (string, error) {
	return "", fmt.Errorf("No space left on device")
}

func (s *noSpaceLeftLocalStorage) GetObjectBytes(directory, fileName string) (*bytes.Buffer, error) {
	return nil, fmt.Errorf("No space left on device")
}

type createdFile struct {
	filePath string
	data     *bytes.Buffer
}

type mockLocalStorage struct {
	createdFilesChan chan createdFile

	// files have content
	//   key: file path [ <pipeline-slug>/<pipeline-processing-id>/<block-slug>/output.<mimetype> ]
	//   value: file content
	files map[string]*bytes.Buffer
}

func (s *mockLocalStorage) ListObjects(bucket string) ([]string, error) {
	return make([]string, 0), nil
}

func (s *mockLocalStorage) PutObject(bucket, filePath string) error {
	return fmt.Errorf("not implemented")
}

func (s *mockLocalStorage) PutObjectBytes(directory string, buffer *bytes.Buffer) (string, error) {
	s.createdFilesChan <- createdFile{
		filePath: directory,
		data:     buffer,
	}
	return "", nil
}

func (s *mockLocalStorage) GetObject(bucket, objectName string, filePath string) (string, error) {
	return "", nil
}

func (s *mockLocalStorage) GetObjectBytes(directory, fileName string) (*bytes.Buffer, error) {
	return bytes.NewBufferString(textContent), nil
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
) map[string]string {
	pipelinesResponse, err := json.Marshal(pipelines)
	suite.Nil(err)

	blocksResponse, err := json.Marshal(blocks)
	suite.Nil(err)

	return map[string]string{
		"/pipelines": string(pipelinesResponse),
		"/blocks":    string(blocksResponse),
	}
}
