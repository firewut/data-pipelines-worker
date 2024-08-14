package blocks

import (
	"net/http"
)

const (
	BlockHTTPId = "http"
)

type BlockHTTP struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
}

func NewBlockHTTP() *BlockHTTP {
	return &BlockHTTP{
		Id:          "get_http_url",
		Name:        "Get HTTP URL",
		Description: "Block to perform request to a URL and get the content",
		Schema: `{
        	"type": "object",
			"properties": {
				"in": {
					"description": "URL to fetch content from"
					"type": "string",
					"format": "url",
				},
				"out": {
					"description": "Content fetched from the URL",
					"type": "object", 
					"properties": {
						"status_code": {
							"description": "Status code of the request",
							"type": "integer",
						},
						"content": {
							"description": "Content fetched from the URL",
							"type": "string"
						}
					}
				}
			}
		}`,
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

func (b *BlockHTTP) GetSchema() string {
	return b.Schema
}

func (b *BlockHTTP) Detect(params ...interface{}) bool {
	if len(params) != 2 {
		return false
	}
	client := params[0]
	testUrl := params[1]

	// Check if the client is an HTTP client
	httpClient, ok := client.(*http.Client)
	if !ok {
		httpClient = &http.Client{}
	}

	_, err := httpClient.Get(testUrl.(string))
	return err == nil
}

func (b *BlockHTTP) Process(data interface{}) int {

	return 0
}
