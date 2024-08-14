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

type BlockSchemaValidator struct {
}

func (b *BlockSchemaValidator) Validate(block Block) (*gojsonschema.Schema, interface{}, error) {
	gojsonschema.FormatCheckers.Add("file", FileFormatChecker{})

	schemaLoader := gojsonschema.NewStringLoader(block.GetSchemaString())
	schemaPtr, err := gojsonschema.NewSchema(schemaLoader)
	schema, _ := schemaLoader.LoadJSON()

	return schemaPtr, schema, err
}

type Block interface {
	GetId() string
	GetName() string
	GetDescription() string
	GetSchemaString() string
	GetSchema() *gojsonschema.Schema

	ValidateSchema(s BlockSchemaValidator) (*gojsonschema.Schema, interface{}, error)
	SetSchema(BlockSchemaValidator) error

	Detect(d BlockDetector) bool
	Process(p BlockProcessor) int
}
