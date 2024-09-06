package unit_test

import (
	"bytes"
	"fmt"
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

func (suite *UnitTestSuite) GetMockHTTPServerURL(body string, statusCode int) string {
	suite.Lock()
	defer suite.Unlock()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if body != "" {
			w.Write([]byte(body))
		}
	}))
	suite.httpTestServers = append(suite.httpTestServers, server)

	return server.URL
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
			"slug": "test",
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

func (suite *UnitTestSuite) GetTestPipelineAndInputForProcessing(
	pipelineSlug string,
	blockSlug string,
	successUrl string,
) (interfaces.Pipeline, schemas.PipelineStartInputSchema) {
	processingData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: pipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: blockSlug,
		},
	}

	pipeline, err := dataclasses.NewPipelineFromBytes([]byte(
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
	))
	suite.Nil(err)
	suite.NotEmpty(pipeline)

	return pipeline, processingData
}

func (suite *UnitTestSuite) RegisterTestPipelineAndInputForProcessing(
	pipelineSlug string,
	blockSlug string,
	successUrl string,
) (interfaces.Pipeline, schemas.PipelineStartInputSchema, *registries.PipelineRegistry) {
	pipeline, processingData := suite.GetTestPipelineAndInputForProcessing(
		pipelineSlug,
		blockSlug,
		successUrl,
	)

	registry, err := registries.NewPipelineRegistry(
		dataclasses.NewPipelineCatalogueLoader(),
	)
	suite.Nil(err)

	registry.Register(pipeline)

	return pipeline, processingData, registry
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
