package blocks

import (
	"net/http"
	"sync"

	"data-pipelines-worker/types"

	"github.com/xeipuuv/gojsonschema"
)

type DetectorHTTP struct {
	Client *http.Client
	Url    string
}

func (d *DetectorHTTP) Detect() bool {
	_, err := d.Client.Get(d.Url)
	return err == nil
}

type ProcessorHTTP struct {
}

func (p *ProcessorHTTP) Process() int {
	return 0
}

type BlockHTTP struct {
	sync.Mutex

	Id           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	SchemaString string               `json:"-"`
	SchemaPtr    *gojsonschema.Schema `json:"-"`
	Schema       interface{}          `json:"schema"`
}

func NewBlockHTTP() *BlockHTTP {
	return &BlockHTTP{
		Id:          "http_request",
		Name:        "Request HTTP Resource",
		Description: "Block to perform request to a URL and save the Response",
		SchemaString: `{
        	"type": "object",
			"properties": {
				"in": {
					"type": "object",
					"description": "Input parameters",
					"properties": {
						"url": {
							"description": "URL to fetch content from",
							"type": "string",
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
						}
					},
					"required": ["url"]
				},
				"out": {
					"description": "Content fetched from the URL",
					"type": ["string", "null"],
					"format": "file"
				}
			}
		}`,
		SchemaPtr: nil,
		Schema:    nil,
	}
}

func (b *BlockHTTP) GetId() string {
	return b.Id
}

func (b *BlockHTTP) GetName() string {
	return b.Name
}

func (b *BlockHTTP) GetDescription() string {
	return b.Description
}

func (b *BlockHTTP) GetSchema() *gojsonschema.Schema {
	b.Lock()
	defer b.Unlock()

	return b.SchemaPtr
}

func (b *BlockHTTP) GetSchemaString() string {
	return b.SchemaString
}

func (b *BlockHTTP) SetSchema(v types.BlockSchemaValidator) error {
	b.Lock()
	defer b.Unlock()

	schemaPtr, schema, err := v.Validate(b)
	if err == nil {
		b.SchemaPtr = schemaPtr
		b.Schema = schema
	}

	return err
}

func (b *BlockHTTP) Detect(d types.BlockDetector) bool {
	b.Lock()
	defer b.Unlock()

	return d.Detect()
}

func (b *BlockHTTP) ValidateSchema(v types.BlockSchemaValidator) (*gojsonschema.Schema, interface{}, error) {
	b.Lock()
	defer b.Unlock()

	return v.Validate(b)
}

func (b *BlockHTTP) Process(p types.BlockProcessor) int {
	return p.Process()
}
