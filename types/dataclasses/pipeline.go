package dataclasses

import (
	"encoding/json"
	"fmt"
	"sync"

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
	logger := config.GetLogger()
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

	// If inputData.Block.TargetIndex is set - Load it's result also
	if inputData.Block.TargetIndex >= 0 {
		pipelineBlockDataRegistry.LoadOutput(inputData.Block.Slug)
	}

	go func() {
		// Loop through each block
		for blockRelativeIndex, blockData := range processBlocks {
			blockIndex := blockRelativeIndex + len(processedBlocks)
			block := blockData.GetBlock()
			if block == nil {
				err := fmt.Errorf(
					"block not found for block data %s",
					blockData.GetSlug(),
				)
				logger.Error(err)
				return
			}

			tmpProcessing := NewProcessing(processingId, p, block, blockData)
			processingRegistry.Add(tmpProcessing)

			var (
				inputConfigValue []map[string]interface{}
				parallel         bool
			)
			blockInputWg := &sync.WaitGroup{}

			if blockData.GetInputConfig() != nil {
				var err error
				inputConfigValue, parallel, err = blockData.GetInputConfigData(
					pipelineBlockDataRegistry.GetAll(),
				)

				if err != nil {

					tmpProcessing.Stop(
						interfaces.ProcessingStatusFailed,
						fmt.Errorf(
							"error getting input config data for block %s [%s:%s]. Error: %s",
							block.GetId(),
							blockData.GetSlug(),
							processingId,
							err,
						),
					)
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

			// Check
			blockInputData = MergeMaps(append(blockInputData, inputConfigValue...))

			// Check registry if Block is Available
			if !blockRegistry.IsAvailable(block) {
				_inputData := schemas.PipelineStartInputSchema{
					Pipeline: schemas.PipelineInputSchema{
						Slug:         blockData.GetPipeline().GetSlug(),
						ProcessingID: processingId,
					},
					Block: schemas.BlockInputSchema{
						Slug:  blockData.GetSlug(),
						Input: make(map[string]interface{}),
					},
				}
				if blockRelativeIndex == 0 {
					_inputData = inputData
				}

				if err := workerRegistry.ResumeProcessing(
					blockData.GetPipeline().GetSlug(),
					processingId,
					block.GetId(),
					_inputData,
				); err != nil {
					tmpProcessing.Stop(interfaces.ProcessingStatusFailed, err)
					return
				}

				tmpProcessing.Stop(interfaces.ProcessingStatusTransferred, nil)
				return
			}

			logger.Infof(
				"Starting processing data for block %s [%s]",
				block.GetId(),
				blockData.GetSlug(),
			)

			type blockInputProcessingResult struct {
				index   int
				err     error
				skipped bool
				stop    bool
			}

			blockInputProcessingResults := make(chan blockInputProcessingResult, len(blockInputData))
			defer close(blockInputProcessingResults)

			for blockInputIndex, blockInput := range blockInputData {
				registryAddBlockData := true
				if blockRelativeIndex == 0 && inputData.Block.TargetIndex >= 0 {
					if blockInputIndex != inputData.Block.TargetIndex {
						blockInputProcessingResults <- blockInputProcessingResult{
							index:   blockInputIndex,
							err:     nil,
							skipped: true,
							stop:    false,
						}

						logger.Infof(
							"Skipping processing data for block %s [%s] with index %d",
							block.GetId(),
							blockData.GetSlug(),
							blockInputIndex,
						)
						continue
					}

					registryAddBlockData = false
				}

				logger.Infof(
					"Processing data for block %s [%s] with index %d",
					block.GetId(),
					blockData.GetSlug(),
					blockInputIndex,
				)

				blockInputWg.Add(1)

				// Clone block data
				blockDataClone := blockData.Clone()
				blockDataClone.SetInputData(blockInput)
				blockDataClone.SetInputIndex(blockInputIndex)

				processing := NewProcessing(processingId, p, block, blockDataClone)

				processBlockInput := func(
					_blockData interfaces.ProcessableBlockData,
					_processing interfaces.Processing,
					_blockInputProcessingResults chan blockInputProcessingResult,
				) error {
					defer blockInputWg.Done()

					processingResult, stopProcessing, err := processingRegistry.StartProcessing(_processing)
					_blockInputProcessingResults <- blockInputProcessingResult{
						index:   blockInputIndex,
						err:     err,
						skipped: false,
						stop:    stopProcessing,
					}

					if err != nil {
						_err := fmt.Errorf(
							"error processing data for block %s [%s:%s] with index %d. Error: %s",
							block.GetId(),
							_blockData.GetSlug(),
							_processing.GetId(),
							blockInputIndex,
							err,
						)
						logger.Error(_err)
						return _err
					}

					logger.Infof(
						"Processing data for block %s [%s:%s] with index %d completed",
						block.GetId(),
						_blockData.GetSlug(),
						_processing.GetId(),
						blockInputIndex,
					)

					if stopProcessing {
						logger.Infof(
							"Pipeline stopped by block %s [%s:%s]",
							block.GetId(),
							_blockData.GetSlug(),
							_processing.GetId(),
						)
						_processing.Stop(interfaces.ProcessingStatusStopped, nil)
						return nil
					}

					logger.Infof(
						"Saving output for block %s [%s:%s] with index %d",
						block.GetId(),
						_blockData.GetSlug(),
						_processing.GetId(),
						blockInputIndex,
					)

					if registryAddBlockData {
						pipelineBlockDataRegistry.AddBlockData(
							_blockData.GetSlug(),
							processingResult.GetValue(),
						)
					} else {
						pipelineBlockDataRegistry.UpdateBlockData(
							_blockData.GetSlug(),
							blockInputIndex,
							processingResult.GetValue(),
						)
					}

					// Save result to Storage
					saveOutputResults := pipelineBlockDataRegistry.SaveOutput(
						_blockData.GetSlug(),
						blockInputIndex,
						processingResult.GetValue(),
					)

					for _, saveOutputResult := range saveOutputResults {
						if saveOutputResult.Error != nil {
							logger.Errorf(
								"Error saving output for block %s [%s] processing %s to storage %s: %s",
								block.GetId(),
								_blockData.GetSlug(),
								_processing.GetData().GetId(),
								saveOutputResult.StorageLocation.GetStorageName(),
								saveOutputResult.Error,
							)
						} else {
							logger.Infof(
								"Saved output for block %s [%s] processing %s to storage %s",
								block.GetId(),
								_blockData.GetSlug(),
								_processing.GetData().GetId(),
								saveOutputResult.StorageLocation.GetStorageName(),
							)
						}
					}

					return nil
				}

				if parallel {
					go processBlockInput(blockDataClone, processing, blockInputProcessingResults)
				} else {
					processBlockInput(blockDataClone, processing, blockInputProcessingResults)
				}
			}

			blockInputWg.Wait()

			for i := 0; i < len(blockInputData); i++ {
				blockInputResult := <-blockInputProcessingResults
				if blockInputResult.err != nil || blockInputResult.stop {
					return
				}
			}
		}

		logger.Infof(
			"Processing Pipeline %s [%s] completed",
			p.GetSlug(),
			processingId,
		)
	}()

	return processingId, nil
}
