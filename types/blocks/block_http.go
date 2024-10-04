package blocks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	gjm "github.com/firewut/go-json-map"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
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
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, error) {
	var output *bytes.Buffer = &bytes.Buffer{}

	logger := config.GetLogger()

	_data := data.GetInputData().(map[string]interface{})

	// Fetch the URL from the input data
	url, err := helpers.GetValue[string](_data, "url")
	if err != nil {
		return nil, false, err
	}

	method := http.MethodGet
	switch m, err := gjm.GetProperty(_data, "method"); {
	case err == nil:
		method = m.(string)
	default:
		logger.Debugf("Method not provided, using default: %s", method)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, false, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second, // Set a timeout for the request ( use config later )
	}

	// Perform the HTTP request
	response, err := client.Do(req)
	if err != nil {
		// Check if the error is due to context cancellation
		if ctx.Err() == context.Canceled {
			logger.Errorf("Request was cancelled for block %s", data.GetSlug())
			return nil, false, ctx.Err()
		}
		return nil, false, err
	}
	defer response.Body.Close()

	_, err = io.Copy(output, response.Body)
	if err != nil {
		return output, false, err
	}

	// Check response status code
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP request failed with status code: %d", response.StatusCode)
		logger.Error(err)
		return output, false, err
	}

	return output, false, nil
}

type BlockHTTPConfig struct {
}

type BlockHTTP struct {
	generics.ConfigurableBlock[BlockHTTPConfig]

	BlockParent
}

var _ interfaces.Block = (*BlockHTTP)(nil)

func (b *BlockHTTP) GetBlockConfig(_config config.Config) *BlockHTTPConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockHTTPConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
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
