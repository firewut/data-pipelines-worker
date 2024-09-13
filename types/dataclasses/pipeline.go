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
	Id          uuid.UUID                         `json:"-"`
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
	registryBlocks := registries.GetBlockRegistry().GetAll()

	// Convert []BlockData to []interfaces.ProcessableBlockData
	p.Blocks = make([]interfaces.ProcessableBlockData, len(aux.Blocks))

	// Loop through each BlockData
	for i, block := range aux.Blocks {
		block.SetPipeline(p)
		block.SetBlock(registryBlocks[block.GetId()])

		p.Blocks[i] = block
	}

	p.Id = uuid.New()
	return nil
}

func NewPipelineFromBytes(data []byte) (*PipelineData, error) {
	pipeline := &PipelineData{}
	err := pipeline.UnmarshalJSON(data)

	return pipeline, err
}

func NewPipelineData() *PipelineData {
	pipeline := &PipelineData{
		Id:     uuid.New(),
		Blocks: make([]interfaces.ProcessableBlockData, 0),
	}

	return pipeline
}

func (p *PipelineData) GetId() string {
	return p.Id.String()
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

func (p *PipelineData) Process(
	data schemas.PipelineStartInputSchema,
	resultStorages []interfaces.Storage,
) (uuid.UUID, error) {
	processingId := uuid.New()
	if data.Pipeline.ProcessingID != nil {
		processingId, _ = uuid.Parse(*data.Pipeline.ProcessingID)
	}

	// Check if the block exists in the pipeline
	processBlocks := make([]interfaces.ProcessableBlockData, 0)
	for i, block := range p.Blocks {
		if block.GetSlug() == data.Block.Slug {
			processBlocks = p.Blocks[i:]
			break
		}
	}

	if len(processBlocks) == 0 {
		return uuid.UUID{}, fmt.Errorf(
			"block with slug %s not found in pipeline %s",
			data.Block.Slug,
			p.Slug,
		)
	}

	// Loop through each block
	for _, blockData := range processBlocks {
		block := blockData.GetBlock()

		if !block.IsAvailable() {
			workerRegistry := registries.GetWorkerRegistry()

			if err := workerRegistry.ResumeProcessing(
				blockData.GetPipeline().GetSlug(),
				processingId,
				blockData.GetSlug(),
			); err != nil {
				return uuid.UUID{}, err
			}
			return processingId, nil
		}

		// Check data.Block.Input not empty
		if len(data.Block.Input) > 0 {
			blockData.SetInputData(data.Block.Input)
		}

		blockProcessor := block.GetProcessor()

		// Start processing
		result, err := block.Process(blockProcessor, blockData)
		if err != nil {
			return uuid.UUID{}, err
		}

		// Save result
		for _, resultStorage := range resultStorages {
			if _, err = block.SaveOutput(
				blockData,
				result,
				processingId,
				resultStorage,
			); err != nil {
				return uuid.UUID{}, err
			}
		}
	}

	return processingId, nil
}
