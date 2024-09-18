package schemas

import (
	"github.com/google/uuid"
)

// Pipeline represents the structure of the pipeline object in the JSON.
type PipelineInputSchema struct {
	Slug         string    `json:"slug"`
	ProcessingID uuid.UUID `json:"processing_id"`
}

// Block represents the structure of the block object in the JSON.
type BlockInputSchema struct {
	Slug  string                 `json:"slug"`
	Input map[string]interface{} `json:"input"`
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
