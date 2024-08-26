package types

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"
)

type PipelineProcessor interface {
	Process() int
}

type Pipeline struct {
	Slug        string        `json:"slug"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Blocks      []BlockStruct `json:"blocks"`

	schemaString string
	schemaPtr    *gojsonschema.Schema
}

func NewPipeline(content []byte) *Pipeline {
	schemaString := string(content)
	v := JSONSchemaValidator{}

	schemaPtr, _, err := v.Validate(schemaString)
	if err != nil {
		panic(err)
	}

	pipeline := &Pipeline{}
	if err := json.Unmarshal(content, pipeline); err != nil {
		panic(err)
	}

	pipeline.schemaString = schemaString
	pipeline.schemaPtr = schemaPtr

	return pipeline
}
