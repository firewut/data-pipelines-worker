package interfaces

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

// BlockDetector represents a detector for a block in a pipeline.
// It provides methods to detect the availability of a block and manage the detection loop.
type BlockDetector interface {

	// Detect performs a detection operation.
	// @return bool True if the detection is successful, false otherwise.
	Detect() bool

	// Start initiates the detection loop.
	// @param block The block to be detected.
	// @param condition A function that returns a boolean indicating whether the detection should continue.
	// @return void
	Start(Block, func() bool)

	// Stop stops the detection loop.
	// @param wg A wait group to synchronize the stopping of the detection loop.
	// @return void
	Stop(*sync.WaitGroup)
}

// BlockProcessor represents a processor for a block in a pipeline.
// It provides methods to process block data.
type BlockProcessor interface {

	// GetRetryCount returns the retry count for the given block.
	// @param block The block for which to get the retry count.
	// @return int The retry count for the block.
	GetRetryCount(Block) int

	// GetRetryInterval returns the retry interval for the given block.
	// @param block The block for which to get the retry interval.
	// @return time.Duration The retry interval for the block.
	GetRetryInterval(Block) time.Duration

	// Process processes the block data using the provided context, block, and data.
	// @param ctx The context for the processing.
	// @param block The block to be processed.
	// @param data The data to be processed.
	// @return *bytes.Buffer The processed data.
	// @return bool A boolean indicating the pipeline must be stopped.
	// @return bool A boolean indicating block processing must be retries.
	// @return string A string indicating block Slug if TargetBlockIndex is > -1.
	// @return int A TargetBlockIndex if processing has Regenerate request. Default -1.
	// @return error An error if the processing fails.
	Process(context.Context, Block, ProcessableBlockData) (
		output *bytes.Buffer,
		stopPipeline bool,
		retryBlockProcessing bool,
		gotoProcessingTargetBlock string,
		gotoProcessingTargetBlockInputIndex int,
		err error,
	)
}

// Block represents a block in a pipeline.
// It provides methods to get and set block properties, schema, processor, and availability status.
type Block interface {

	// GetId returns the unique identifier of the block.
	// @return string The unique identifier of the block.
	GetId() string

	// GetName returns the name of the block.
	// @return string The name of the block.
	GetName() string

	// GetDescription returns the description of the block.
	// @return string The description of the block.
	GetDescription() string

	// GetVersion returns the version of the block.
	// @return string The version of the block.
	GetVersion() string

	// SetSchemaString sets the schema string for the block.
	// @param schema The schema string to be set.
	// @return void
	SetSchemaString(string)

	// GetSchemaString returns the schema string of the block.
	// @return string The schema string of the block.
	GetSchemaString() string

	// GetSchema returns the schema of the block.
	// @return *gojsonschema.Schema The schema of the block.
	GetSchema() *gojsonschema.Schema

	// ApplySchema applies the schema to the block.
	// @param schema The schema string to be applied.
	// @return error An error if the schema application fails.
	ApplySchema(string) error

	// SetAvailable sets the availability status of the block.
	// @param available The availability status to be set.
	// @return void
	SetAvailable(bool)

	// GetAvailable returns the availability status of the block.
	// @return bool The availability status of the block.
	GetAvailable() bool

	// IsAvailable checks if the block is available.
	// @return bool True if the block is available, false otherwise.
	IsAvailable() bool

	// SetProcessor sets the processor for the block.
	// @param processor The processor to be set.
	// @return void
	SetProcessor(BlockProcessor)

	// GetProcessor returns the processor of the block.
	// @return BlockProcessor The processor of the block.
	GetProcessor() BlockProcessor

	// Process processes the block data using the provided context, processor, and data.
	// @param ctx The context for the processing.
	// @param processor The processor to be used for processing.
	// @param data The data to be processed.
	// @return *bytes.Buffer The processed data.
	// @return bool A boolean indicating the pipeline must be stopped.
	// @return bool A boolean indicating block processing must be retries.
	// @return string A string indicating block Slug if TargetBlockIndex is > -1.
	// @return int A TargetBlockIndex if processing has Regenerate request. Default -1.
	// @return error An error if the processing fails.
	Process(
		context.Context,
		BlockProcessor,
		ProcessableBlockData,
	) (
		output *bytes.Buffer,
		stopPipeline bool,
		retryBlockProcessing bool,
		gotoProcessingTargetBlock string,
		gotoProcessingTargetBlockInputIndex int,
		err error,
	)
}

// ProcessableBlockData represents a data structure that can be processed in a pipeline.
// It provides methods to get and set block data and handle various pipeline input configurations.
type ProcessableBlockData interface {

	// SetPipeline sets the pipeline that processes the block data.
	// @param pipeline The pipeline to be set.
	// @return void
	SetPipeline(Pipeline)

	// GetPipeline retrieves the current pipeline associated with the block data.
	// @return The current pipeline.
	GetPipeline() Pipeline

	// SetBlock sets the block that is part of the pipeline.
	// @param block The block to be set.
	// @return void
	SetBlock(Block)

	// GetBlock retrieves the current block associated with the pipeline.
	// @return The current block.
	GetBlock() Block

	// GetId returns the unique identifier for the block.
	// @return The unique identifier for the block.
	GetId() string

	// GetSlug retrieves the slug associated with the block.
	// @return The slug for the block.
	GetSlug() string

	// GetInputIndex retrieves the input index for the block data.
	// @return The input index.
	GetInputIndex() int

	// SetInputIndex sets the input index for the block data.
	// @param index The input index to be set.
	// @return void
	SetInputIndex(int)

	// GetInput retrieves the input data for the block as a map.
	// @return The input data as a map of string keys to interface values.
	GetInput() map[string]interface{}

	// GetInputConfig retrieves the configuration for the input data.
	// @return The input configuration as a map.
	GetInputConfig() map[string]interface{}

	// GetData retrieves the data associated with the block.
	// @return The block data.
	GetData() interface{}

	// SetData sets the data for the block.
	// @param data The data to be set.
	// @return void
	SetData(interface{})

	// GetInputData retrieves the input data for the block.
	// @return The input data.
	GetInputData() interface{}

	// SetInputData sets the input data for the block.
	// @param data The input data to be set.
	// @return void
	SetInputData(interface{})

	// GetInputDataByPriority retrieves input data based on specified priorities.
	// @param priorities A slice of interfaces representing priorities.
	// @return A slice of maps containing the input data based on the priorities.
	GetInputDataByPriority([]interface{}) []map[string]interface{}

	// GetInputConfigData retrieves input configuration data based on provided buffers.
	// @param buffers A map of string keys to slice of bytes buffers.
	// @return The input configuration data as a slice of maps, a bool indicating success, and an error if any.
	GetInputConfigData(map[string][]*bytes.Buffer) ([]map[string]interface{}, bool, error)

	// GetStringRepresentation returns a string representation of the block data.
	// @return The string representation of the data.
	GetStringRepresentation() string

	// Clone creates a copy of the ProcessableBlockData.
	// @return A clone of the current ProcessableBlockData.
	Clone() ProcessableBlockData
}
