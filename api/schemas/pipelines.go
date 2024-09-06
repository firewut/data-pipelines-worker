package schemas

// Pipeline represents the structure of the pipeline object in the JSON.
type PipelineInputSchema struct {
	Slug         string  `json:"slug"`
	ProcessingID *string `json:"processing_id"` // Use a pointer to handle null values
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
