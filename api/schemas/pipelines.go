package schemas

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// PipelineInputSchema represents the structure of the pipeline input object in the JSON.
// It contains information about the pipeline's slug and processing ID.
//
// swagger:model
type PipelineInputSchema struct {
	// The slug of the pipeline
	// required: true
	// example: "example-slug"
	Slug string `json:"slug"`

	// The unique processing ID associated with this pipeline
	// required: true
	// example: "d9b2d63d5f23e4d76b7f3f2f25d93a7a"
	ProcessingID uuid.UUID `json:"processing_id"`
}

// BlockInputSchema represents the structure of the block object in the JSON.
// It contains information about the block's slug, its input data, target index,
// and an optional destination slug.
//
// swagger:model
type BlockInputSchema struct {
	// The slug of the block
	// required: true
	// example: "example-block"
	Slug string `json:"slug"`

	// The input data for the block, represented as a map of key-value pairs
	// required: true
	// example: {"key1": "value1", "key2": 42}
	Input map[string]interface{} `json:"input"`

	// The target index for the block (optional)
	// If omitted or empty, it defaults to -1
	// example: 5
	TargetIndex int `json:"target_index,omitempty,string"`

	// The destination slug for the block (optional)
	// example: "destination-block"
	DestinationSlug string `json:"destination_slug,omitempty"`
}

// UnmarshalJSON for BlockInputSchema to handle empty string for TargetIndex
func (b *BlockInputSchema) UnmarshalJSON(data []byte) error {
	type Alias BlockInputSchema
	aux := &struct {
		TargetIndex *string `json:"target_index"` // Pointer to string for handling empty value
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Check if TargetIndex is nil (not present) or an empty string
	if aux.TargetIndex == nil || *aux.TargetIndex == "" {
		b.TargetIndex = -1 // Set to -1 if nil or empty
	} else {
		// Convert the string to an int
		var value int
		if _, err := fmt.Sscanf(*aux.TargetIndex, "%d", &value); err != nil {
			return fmt.Errorf("invalid target_index: %v", err)
		}
		b.TargetIndex = value
	}

	return nil
}

// PipelineStartInputSchema represents the structure of the entire JSON payload.
// It contains the `Pipeline` and `Block` data, which are used to start the pipeline process.
//
// swagger:model
type PipelineStartInputSchema struct {
	// The pipeline information, represented by the PipelineInputSchema model
	// required: true
	Pipeline PipelineInputSchema `json:"pipeline"`

	// The block information, represented by the BlockInputSchema model
	// required: true
	Block BlockInputSchema `json:"block"`
}

// GetProcessingID returns the processing ID for the pipeline.
// If the Pipeline has a valid ProcessingID, it returns that, otherwise, it generates a new one.
//
// This method ensures that each pipeline process is associated with a unique ProcessingID.
// If the ProcessingID is missing, a new one is created and assigned to the Pipeline.
func (p *PipelineStartInputSchema) GetProcessingID() uuid.UUID {
	processingId := uuid.New()

	if p.Pipeline.ProcessingID != uuid.Nil {
		processingId = p.Pipeline.ProcessingID
	} else {
		p.Pipeline.ProcessingID = processingId
	}

	return processingId
}

// PipelineStartOutputSchema represents the structure of the output JSON
// when a new pipeline is started. It contains the unique processing ID
// that is generated or provided for the pipeline.
//
// swagger:model
type PipelineStartOutputSchema struct {
	// The unique processing ID for the pipeline
	// required: true
	// example: "d9b2d63d-5f23-e4d7-6b7f-3f2f25d93a7a"
	ProcessingID uuid.UUID `json:"processing_id"`
}

// PipelineResumeOutputSchema represents the structure of the output JSON
// when resuming a pipeline. It includes the processing ID associated
// with the pipeline that was previously started.
//
// swagger:model
type PipelineResumeOutputSchema struct {
	// The unique processing ID for the resumed pipeline
	// required: true
	// example: "d9b2d63d-5f23-e4d7-6b7f-3f2f25d93a7a"
	ProcessingID uuid.UUID `json:"processing_id"`
}
