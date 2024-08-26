package types

import (
	"github.com/xeipuuv/gojsonschema"
)

type BlockDetector interface {
	Detect() bool
}

type BlockProcessor interface {
	Process() int
}

type Block interface {
	GetId() string
	GetName() string
	GetDescription() string
	GetSchemaString() string
	GetSchema() *gojsonschema.Schema

	ValidateSchema(s JSONSchemaValidator) (*gojsonschema.Schema, interface{}, error)
	SetSchema(JSONSchemaValidator) error

	Detect(d BlockDetector) bool
	Process(p BlockProcessor) int
}

// JSON will parse to this data structure
type BlockStruct struct {
	Id           string                 `json:"id"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description"`
	Inputconfig  map[string]interface{} `json:"input_config"`
	Input        map[string]interface{} `json:"input"`
	OutputConfig map[string]interface{} `json:"output_config"`
}
