package unit_test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"

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
	_config         config.Config
	_detectedBlocks map[string]interfaces.Block
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
	suite._detectedBlocks = make(map[string]interfaces.Block)
}

func (suite *UnitTestSuite) TearDownSuite() {
}

func (suite *UnitTestSuite) SetupTest() {

}

func (suite *UnitTestSuite) TearDownTest() {
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
