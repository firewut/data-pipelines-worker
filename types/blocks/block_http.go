package blocks

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	gjm "github.com/firewut/go-json-map"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

type DetectorHTTP struct {
	BlockDetectorParent

	Client *http.Client
	Url    string
}

func NewDetectorHTTP(
	client *http.Client,
	detectorConfig config.BlockConfigDetector,
) *DetectorHTTP {
	return &DetectorHTTP{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
		Client:              client,
		Url:                 detectorConfig.Conditions["url"].(string),
	}
}

func (d *DetectorHTTP) Detect() bool {
	d.Lock()
	defer d.Unlock()

	_, err := d.Client.Get(d.Url)
	return err == nil
}

type ProcessorHTTP struct {
}

func NewProcessorHTTP() *ProcessorHTTP {
	return &ProcessorHTTP{}
}

func (p *ProcessorHTTP) Process(
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, error) {
	var output *bytes.Buffer = &bytes.Buffer{}

	logger := config.GetLogger()

	_data := data.GetInputData().(map[string]interface{})

	url, err := gjm.GetProperty(_data, "url")
	if err != nil {
		return nil, err
	}

	response, err := http.Get(url.(string))
	if err != nil {
		logger.Errorf("HTTP request error: %v", err)
		return output, err
	}
	defer response.Body.Close()

	// Read response body
	_, err = io.Copy(output, response.Body)
	if err != nil {
		return output, err
	}

	// Check response status code
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP request failed with status code: %d", response.StatusCode)
		logger.Error(err)
		return output, err
	}

	return output, nil
}

type BlockHTTP struct {
	BlockParent
}

func NewBlockHTTP() *BlockHTTP {
	block := &BlockHTTP{
		BlockParent: BlockParent{
			Id:          "http_request",
			Name:        "Request HTTP Resource",
			Description: "Block to perform request to a URL and save the Response",
			Version:     "1",
			SchemaString: `{
        	"type": "object",
			"properties": {
				"input": {
					"type": "object",
					"description": "Input parameters",
					"properties": {
						"url": {
							"description": "URL to fetch content from",
							"type": "string",
							"minLength": 5,
							"format": "url"
						},
						"method": {
							"description": "HTTP method to use",
							"type": "string",
							"enum": ["GET", "POST", "PUT", "DELETE"],
							"default": "GET"
						},
						"headers": {
							"description": "HTTP headers to send",
							"type": ["object", "null"],
							"additionalProperties": {
								"type": "string"
							},
							"example": {
								"Content-Type": "application/json"
							}
						},
						"query": {
							"description": "Query parameters to send in the request",
							"type": ["object", "null"],
							"additionalProperties": {
								"type": "string"
							},
							"example": {
								"page": "1",
								"limit": "10"
							}
						},
						"body": {
							"description": "Body to send in the request",
							"type": ["string", "object", "array", "null"]
						},
						"description": {
							"description": "Description of the request",
							"type": ["string", "null"]
						}
					},
					"required": ["url"]
				},
				"output": {
					"description": "Content fetched from the URL",
					"type": ["string", "null"],
					"format": "file"
				}
			}
		}`,
			SchemaPtr: nil,
			Schema:    nil,
		},
	}

	if err := block.ApplySchema(block.GetSchemaString()); err != nil {
		panic(err)
	}

	block.SetProcessor(NewProcessorHTTP())

	return block
}
