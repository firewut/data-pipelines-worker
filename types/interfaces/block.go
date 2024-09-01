package interfaces

import (
	"bytes"

	"github.com/xeipuuv/gojsonschema"
)

type BlockDetector interface {
	Detect() bool
}

type BlockProcessor interface {
	Process(Block, ProcessableBlockData) (*bytes.Buffer, error)
}

type Block interface {
	GetId() string
	GetName() string
	GetDescription() string
	GetSchemaString() string
	GetSchema() *gojsonschema.Schema

	ApplySchema(string) error

	SetAvailable(bool)
	IsAvailable() bool
	Detect(BlockDetector) bool

	Process(BlockProcessor, ProcessableBlockData) (*bytes.Buffer, error)
	SaveOutput(ProcessableBlockData, *bytes.Buffer, Storage) (string, error)
}

type ProcessableBlockData interface {
	SetPipeline(Pipeline)
	GetPipeline() Pipeline

	SetBlock(Block)
	GetBlock() Block

	GetId() string
	GetSlug() string

	GetData() interface{}
	SetData(interface{})

	GetInputData() interface{}

	GetStringRepresentation() string
}
