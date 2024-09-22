package dataclasses

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/config"
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
		registryBlock := registryBlocks[block.GetId()]
		block.SetPipeline(p)
		block.SetBlock(registryBlock)

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
	workerRegistry interfaces.WorkerRegistry,
	blockRegistry interfaces.BlockRegistry,
	processingRegistry interfaces.ProcessingRegistry,
	inputData schemas.PipelineStartInputSchema,
	resultStorages []interfaces.Storage,
) (uuid.UUID, error) {
	processingId := inputData.GetProcessingID()

	// Check if the block exists in the pipeline
	pipelineBlocks := p.GetBlocks()
	processedBlocks := make([]interfaces.ProcessableBlockData, 0)
	processBlocks := make([]interfaces.ProcessableBlockData, 0)
	for i, block := range pipelineBlocks {
		if block.GetSlug() == inputData.Block.Slug {
			processedBlocks = p.Blocks[:i]
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

	pipelineBlockDataRegistry := registries.NewPipelineBlockDataRegistry(
		processingId,
		p.Slug,
		resultStorages,
	)

	// Prepare results of previous Pipeline execution
	// e.g. if previous Worker passed request to this worker
	for _, blockData := range processedBlocks {
		pipelineBlockDataRegistry.LoadOutput(blockData.GetSlug())
	}

	go func() {
		// Loop through each block
		for blockRelativeIndex, blockData := range processBlocks {
			blockIndex := blockRelativeIndex + len(processedBlocks)

			block := blockData.GetBlock()

			tmpProcessing := NewProcessing(processingId, p, block, blockData)
			processingRegistry.Add(tmpProcessing)

			// Check registry if Block is Available
			if !blockRegistry.IsAvailable(block) {
				if err := workerRegistry.ResumeProcessing(
					blockData.GetPipeline().GetSlug(),
					processingId,
					blockData.GetId(),
					inputData,
				); err != nil {
					tmpProcessing.Stop(interfaces.ProcessingStatusFailed, err)
					return
				}

				tmpProcessing.Stop(interfaces.ProcessingStatusTransferred, nil)
				return
			}

			var inputConfigValue interface{}
			if blockData.GetInputConfig() != nil {
				var err error
				inputConfigValue, err = blockData.GetInputConfigData(
					pipelineBlockDataRegistry.GetAll(),
				)
				if err != nil {
					tmpProcessing.Stop(interfaces.ProcessingStatusFailed, err)
					return
				}
			}

			blockInputData := blockData.GetInputDataByPriority(
				[]interface{}{
					BlockInputData{
						Condition: (blockRelativeIndex == 0 ||
							blockIndex == 0) &&
							inputData.Block.Input != nil &&
							len(inputData.Block.Input) > 0,
						Value: []map[string]interface{}{
							inputData.Block.Input,
						},
					},
					BlockInputData{
						Condition: blockData.GetInputConfig() != nil,
						Value:     inputConfigValue,
					},
					BlockInputData{
						Condition: blockData.GetInputData() != nil,
						Value: []map[string]interface{}{
							blockData.GetInputData().(map[string]interface{}),
						},
					},
				},
			)

			for blockInputIndex, blockInput := range blockInputData {
				blockData.SetInputData(blockInput)

				processing := NewProcessing(processingId, p, block, blockData)
				processingResult, err := processingRegistry.StartProcessing(processing)
				if err != nil {
					return
				}

				pipelineBlockDataRegistry.AddBlockData(
					processingResult.GetId(),
					processingResult.GetValue(),
				)

				// Save result to Storage
				saveOutputResults := pipelineBlockDataRegistry.SaveOutput(
					processing.GetData().GetSlug(),
					blockInputIndex,
					processingResult.GetValue(),
				)

				for _, saveOutputResult := range saveOutputResults {
					if saveOutputResult.Error != nil {
						config.GetLogger().Errorf(
							"Error saving output for block %s to storage %s: %s",
							processing.GetData().GetSlug(),
							saveOutputResult.StorageLocation.GetStorageName(),
							saveOutputResult.Error,
						)
					}
				}
			}
		}
	}()

	return processingId, nil
}
