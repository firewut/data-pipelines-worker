package interfaces

import (
	"bytes"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

type BlockDetector interface {
	Detect() bool             // Performs a detection operation.
	Start(Block, func() bool) // Initiates the detection loop.
	Stop(wg *sync.WaitGroup)  // Stops the detection loop.
}

type BlockProcessor interface {
	Process(Block, ProcessableBlockData) (*bytes.Buffer, error)
}

type Block interface {
	GetId() string
	GetName() string
	GetDescription() string
	GetVersion() string
	GetAvailable() bool
	GetSchemaString() string
	GetSchema() *gojsonschema.Schema
	ApplySchema(string) error

	SetAvailable(bool)
	IsAvailable() bool

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
