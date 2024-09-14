package dataclasses

import (
	"bytes"
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
	inputData schemas.PipelineStartInputSchema,
	resultStorages []interfaces.Storage,
) (uuid.UUID, error) {
	processingId := uuid.New()

	if inputData.Pipeline.ProcessingID != uuid.Nil {
		processingId = inputData.Pipeline.ProcessingID
	} else {
		inputData.Pipeline.ProcessingID = processingId
	}

	// Check if the block exists in the pipeline
	processBlocks := make([]interfaces.ProcessableBlockData, 0)
	for i, block := range p.Blocks {
		if block.GetSlug() == inputData.Block.Slug {
			processBlocks = p.Blocks[i:]
			break
		}
	}

	if len(processBlocks) == 0 {
		return uuid.UUID{}, fmt.Errorf(
			"block with slug %s not found in pipeline %s",
			inputData.Block.Slug,
			p.Slug,
		)
	}

	// Visit Shared Storage and Fetch State to pipelineResults
	pipelineResults := make(map[string][]*bytes.Buffer, 0)

	// Loop through each block
	for blockIndex, blockData := range processBlocks {
		block := blockData.GetBlock()

		if !block.IsAvailable() {
			workerRegistry := registries.GetWorkerRegistry()

			if err := workerRegistry.ResumeProcessing(
				blockData.GetPipeline().GetSlug(),
				processingId,
				blockData.GetSlug(),
				blockIndex,
				inputData,
			); err != nil {
				return uuid.UUID{}, err
			}
			return processingId, nil
		}

		// Priority of Inputs
		var blockInputData = make([]map[string]interface{}, 0)

		// Not Important - get from GetInputData
		if blockData.GetInputData() != nil {
			blockInputData = []map[string]interface{}{
				blockData.GetInputData().(map[string]interface{}),
			}
		}

		// Important - get from `input_config`
		if blockData.GetInputConfig() != nil {
			blockInputData = blockData.GetInputDataFromConfig(
				pipelineResults,
			)
		}

		// Critical - get from request ( function argument `inputData`` )
		if blockIndex == 0 && inputData.Block.Input != nil {
			blockInputData = []map[string]interface{}{
				inputData.Block.Input,
			}
		}

		pipelineResult := make([]*bytes.Buffer, 0)
		for _, blockInput := range blockInputData {
			// Check data.Block.Input not empty
			blockProcessor := block.GetProcessor()

			// Start processing
			blockData.SetInputData(blockInput)
			result, err := block.Process(
				blockProcessor,
				blockData,
			)
			if err != nil {
				return uuid.UUID{}, err
			}

			// Save result
			for _, resultStorage := range resultStorages {
				if _, err = block.SaveOutput(
					blockData,
					result,
					blockIndex,
					processingId,
					resultStorage,
				); err != nil {
					return uuid.UUID{}, err
				}
			}

			pipelineResult = append(pipelineResult, result)
		}

		pipelineResults[blockData.GetSlug()] = pipelineResult
	}

	return processingId, nil
}
