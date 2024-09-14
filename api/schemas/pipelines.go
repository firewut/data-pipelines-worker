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

// func (p PipelineStartInputSchema) DeepCopy() (PipelineStartInputSchema, error) {
// 	var destination PipelineStartInputSchema

// 	jsonContent, err := json.Marshal(p)
// 	if err != nil {
// 		return destination, err
// 	}
// 	err = json.Unmarshal(jsonContent, &destination)
// 	if err != nil {
// 		return destination, err
// 	}

// 	return destination, nil
// }
