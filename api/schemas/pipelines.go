package schemas

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Pipeline represents the structure of the pipeline object in the JSON.
type PipelineInputSchema struct {
	Slug         string    `json:"slug"`
	ProcessingID uuid.UUID `json:"processing_id"`
}

// Block represents the structure of the block object in the JSON.
type BlockInputSchema struct {
	Slug            string                 `json:"slug"`
	Input           map[string]interface{} `json:"input"`
	TargetIndex     int                    `json:"target_index,omitempty,string"`
	DestinationSlug string                 `json:"destination_slug,omitempty"`
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

// PipelineStartInputSchema represents the structure of the entire JSON.
type PipelineStartInputSchema struct {
	Pipeline PipelineInputSchema `json:"pipeline"`
	Block    BlockInputSchema    `json:"block"`
}

func (p *PipelineStartInputSchema) GetProcessingID() uuid.UUID {
	processingId := uuid.New()

	if p.Pipeline.ProcessingID != uuid.Nil {
		processingId = p.Pipeline.ProcessingID
	} else {
		p.Pipeline.ProcessingID = processingId
	}

	return processingId
}

// PipelineStartOutputSchema represents the structure of the output JSON.
type PipelineStartOutputSchema struct {
	ProcessingID uuid.UUID `json:"processing_id"`
}

// PipelineResumeOutputSchema represents the structure of the input JSON.
type PipelineResumeOutputSchema struct {
	ProcessingID uuid.UUID `json:"processing_id"`
}
