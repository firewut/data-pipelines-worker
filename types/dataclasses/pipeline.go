package dataclasses

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
	"data-pipelines-worker/types/validators"
)

type PipelineData struct {
	ID          uuid.UUID                         `json:"-"`
	Slug        string                            `json:"slug"`
	Title       string                            `json:"title"`
	Description string                            `json:"description"`
	Blocks      []interfaces.ProcessableBlockData `json:"blocks"`

	schemaString string
	schemaPtr    *gojsonschema.Schema
}

func (p *PipelineData) UnmarshalJSON(data []byte) error {
	// Declare Blocks a BlockData array
	type Alias PipelineData
	aux := &struct {
		Blocks []*BlockData `json:"blocks"`
		*Alias
	}{
		Blocks: make([]*BlockData, 0),
		Alias:  (*Alias)(p),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	aux.schemaString = string(data)

	v := validators.JSONSchemaValidator{}
	schemaPtr, _, err := v.ValidateSchema(aux.schemaString)
	if err != nil {
		return err
	}
	p.schemaString = aux.schemaString
	p.schemaPtr = schemaPtr

	// List of Blocks
	registryBlocks := registries.GetBlockRegistry().GetBlocks()

	// Convert []BlockData to []interfaces.ProcessableBlockData
	p.Blocks = make([]interfaces.ProcessableBlockData, len(aux.Blocks))

	// Loop through each BlockData
	for i, block := range aux.Blocks {
		block.SetPipeline(p)
		block.SetBlock(registryBlocks[block.GetId()])

		p.Blocks[i] = block
	}

	p.ID = uuid.New()
	return nil
}

func NewPipelineFromBytes(data []byte) (*PipelineData, error) {
	pipeline := &PipelineData{}
	err := pipeline.UnmarshalJSON(data)

	return pipeline, err
}

func NewPipelineData() *PipelineData {
	pipeline := &PipelineData{
		ID:     uuid.New(),
		Blocks: make([]interfaces.ProcessableBlockData, 0),
	}

	return pipeline
}

func (p *PipelineData) GetID() uuid.UUID {
	return p.ID
}

func (p *PipelineData) GetSlug() string {
	return p.Slug
}

func (p *PipelineData) GetTitle() string {
	return p.Title
}

func (p *PipelineData) GetDescription() string {
	return p.Description
}

func (p *PipelineData) GetBlocks() []interfaces.ProcessableBlockData {
	return p.Blocks
}

func (p *PipelineData) GetSchemaString() string {
	return p.schemaString
}

func (p *PipelineData) GetSchemaPtr() *gojsonschema.Schema {
	return p.schemaPtr
}

func (p *PipelineData) StartProcessing(
	data schemas.PipelineStartInputSchema,
	storage interfaces.Storage,
) (uuid.UUID, error) {
	// Check if the block exists in the pipeline
	var blockData interfaces.ProcessableBlockData
	for _, block := range p.Blocks {
		if block.GetSlug() == data.Block.Slug {
			blockData = block
			break
		}
	}

	if blockData == nil {
		return uuid.UUID{}, fmt.Errorf(
			"block with slug %s not found in pipeline %s",
			data.Block.Slug,
			p.Slug,
		)
	}

	// Check data.Block.Input not empty
	if len(data.Block.Input) > 0 {
		blockData.SetInputData(data.Block.Input)
	}

	block := blockData.GetBlock()
	blockProcessor := block.GetProcessor()

	// Start processing
	result, err := block.Process(blockProcessor, blockData)
	if err != nil {
		return uuid.UUID{}, err
	}

	// Save result
	if _, err = block.SaveOutput(blockData, result, storage); err != nil {
		return uuid.UUID{}, err
	}

	return uuid.New(), nil
}
