package interfaces

import (
	"bytes"
	"context"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

type BlockDetector interface {
	Detect() bool             // Performs a detection operation.
	Start(Block, func() bool) // Initiates the detection loop.
	Stop(wg *sync.WaitGroup)  // Stops the detection loop.
}

type BlockProcessor interface {
	Process(context.Context, Block, ProcessableBlockData) (*bytes.Buffer, error)
}

type Block interface {
	GetId() string
	GetName() string
	GetDescription() string
	GetVersion() string
	GetSchemaString() string
	GetSchema() *gojsonschema.Schema
	ApplySchema(string) error

	SetAvailable(bool)
	GetAvailable() bool
	IsAvailable() bool

	SetProcessor(BlockProcessor)
	GetProcessor() BlockProcessor

	Process(context.Context, BlockProcessor, ProcessableBlockData) (*bytes.Buffer, error)
}

type ProcessableBlockData interface {
	SetPipeline(Pipeline)
	GetPipeline() Pipeline

	SetBlock(Block)
	GetBlock() Block

	GetId() string
	GetSlug() string

	GetInputConfig() map[string]interface{}

	GetData() interface{}
	SetData(interface{})

	GetInputData() interface{}
	SetInputData(interface{})

	GetInputDataByPriority([]interface{}) []map[string]interface{}
	GetInputConfigData(map[string][]*bytes.Buffer) ([]map[string]interface{}, error)

	GetStringRepresentation() string
}
