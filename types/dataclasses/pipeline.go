package dataclasses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
	"data-pipelines-worker/types/validators"
)

// PipelineData represents the structure of a pipeline in the system.
// It includes the pipeline's metadata, description, and associated blocks.
//
// swagger:model
type PipelineData struct {
	// The unique identifier for the pipeline
	// required: true
	Id uuid.UUID `json:"-"`

	// The unique slug identifier for the pipeline
	// required: true
	// example: "pipeline-abc123"
	Slug string `json:"slug"`

	// The title of the pipeline
	// required: true
	// example: "Example Pipeline"
	Title string `json:"title"`

	// The description of the pipeline
	// example: "This pipeline processes data in a series of blocks."
	Description string `json:"description"`

	// A list of blocks that make up the pipeline
	// required: true
	Blocks []interfaces.ProcessableBlockData `json:"blocks"`

	// internal field for storing the schema string
	schemaString string

	// internal field for storing the parsed schema pointer
	schemaPtr *gojsonschema.Schema
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
	logger, loggerBuffer := config.GetLoggerForEntity("pipeline", processingId)

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

	// Calculate block Index for inputData.Block.DestinationSlug
	destinationBlockIndex := -1
	if len(inputData.Block.DestinationSlug) > 0 {
		for i, block := range pipelineBlocks {
			if block.GetSlug() == inputData.Block.DestinationSlug {
				destinationBlockIndex = i
				break
			}
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
		blockInputsData := make(map[string][]map[string]interface{}, 0)

		// Save result of Pipeline execution in any case
		defer func() {
			pipelineBlockDataRegistry.SavePipelineLog(loggerBuffer)
		}()

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

			var (
				inputConfigValue []map[string]interface{}
				parallel         bool
			)

			blockInputWg := &sync.WaitGroup{}
			blockCtx, blockCtxCancel := context.WithCancel(context.Background())
			defer blockCtxCancel()

			tmpProcessing := NewProcessing(
				blockCtx,
				blockCtxCancel,
				processingId,
				p,
				block,
				blockData,
			)
			processingRegistry.Add(tmpProcessing)

			if blockData.GetInputConfig() != nil {
				var err error
				inputConfigValue, parallel, err = blockData.GetInputConfigData(
					pipelineBlockDataRegistry.GetAll(),
				)

				if err != nil {
					logger.Error(err)
					tmpProcessing.Stop(
						interfaces.ProcessingStatusFailed,
						fmt.Errorf(
							"error getting input config data for block [%s:%s]. Error: %s",
							blockData.GetSlug(),
							block.GetId(),
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

			// Merge inputs
			blockInputData = MergeMaps(append(blockInputData, inputConfigValue...))

			// Append to local mapping for blockInputsData
			blockInputsData[blockData.GetSlug()] = blockInputData

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
				"Starting processing data for block [%s:%s]",
				blockData.GetSlug(),
				block.GetId(),
			)

			type blockInputProcessingResult struct {
				index   int
				err     error
				skipped bool
				stop    bool
			}

			blockInputProcessingResults := make(chan blockInputProcessingResult, len(blockInputData))
			defer close(blockInputProcessingResults)

			pipelineBlockDataRegistry.PrepareBlockData(blockData.GetSlug(), len(blockInputData))

			for blockInputIndex, blockInput := range blockInputData {
				if inputData.Block.TargetIndex >= 0 {
					if blockRelativeIndex == 0 || (destinationBlockIndex >= 0 && blockIndex < destinationBlockIndex) {
						if blockInputIndex != inputData.Block.TargetIndex {
							blockInputProcessingResults <- blockInputProcessingResult{
								index:   blockInputIndex,
								err:     nil,
								skipped: true,
								stop:    false,
							}

							logger.Infof(
								"Skipping processing data for block [%s:%s] with index %d",
								blockData.GetSlug(),
								block.GetId(),
								blockInputIndex,
							)
							continue
						}
					}
				}

				logger.Infof(
					"Processing data for block [%s:%s] with index %d",
					blockData.GetSlug(),
					block.GetId(),
					blockInputIndex,
				)

				blockInputWg.Add(1)

				// Clone block data
				blockDataClone := blockData.Clone()
				blockDataClone.SetInputData(blockInput)
				blockDataClone.SetInputIndex(blockInputIndex)

				processing := NewProcessing(
					blockCtx,
					blockCtxCancel,
					processingId,
					p,
					block,
					blockDataClone,
				)

				processBlockInput := func(
					_blockData interfaces.ProcessableBlockData,
					_processing interfaces.Processing,
					_blockInputProcessingResults chan blockInputProcessingResult,
				) (interfaces.ProcessingOutput, error) {
					defer blockInputWg.Done()

					processingOutput := processingRegistry.StartProcessing(_processing)
					_blockInputProcessingResults <- blockInputProcessingResult{
						index:   blockInputIndex,
						err:     processingOutput.GetError(),
						skipped: false,
						stop:    processingOutput.GetStop(),
					}

					if processingOutput.GetError() != nil {
						_err := fmt.Errorf(
							"error processing data for block [%s:%s] with index %d. Error: %s",
							_blockData.GetSlug(),
							block.GetId(),
							blockInputIndex,
							processingOutput.GetError(),
						)
						if processingOutput.GetError() != context.Canceled {
							logger.Error(_err)
						}
						return processingOutput, _err
					}

					logger.Infof(
						"Processing data for block [%s:%s] with index %d completed",
						_blockData.GetSlug(),
						block.GetId(),
						blockInputIndex,
					)

					if processingOutput.GetStop() {
						destinationBlockSlug := blockDataClone.GetSlug()
						targetBlockSlug := processingOutput.GetTargetBlockSlug()
						targetBlockInputIndex := processingOutput.GetTargetBlockInputIndex()

						// Call cancel to stop the current block processing
						blockCtxCancel()

						if targetBlockSlug != "" &&
							targetBlockInputIndex >= 0 {
							logger.Warnf(
								"Pipeline stopped by block [%s:%s] with index %d. Regenerating block %s with index %d",
								_blockData.GetSlug(),
								block.GetId(),
								blockInputIndex,
								targetBlockSlug,
								targetBlockInputIndex,
							)

							go func(
								p *PipelineData,
								_workerRegistry interfaces.WorkerRegistry,
								_blockRegistry interfaces.BlockRegistry,
								_processingRegistry interfaces.ProcessingRegistry,
								_resultStorages []interfaces.Storage,
								_targetBlockSlug string,
								_targetBlockInputIndex int,
								_destinationBlockSlug string,
							) {
								input := map[string]interface{}{}
								if inputAtIndex, exists := blockInputsData[_targetBlockSlug]; exists {
									if _targetBlockInputIndex < len(inputAtIndex) {
										input = inputAtIndex[_targetBlockInputIndex]
									}
								}

								regenerateData := schemas.PipelineStartInputSchema{
									Pipeline: schemas.PipelineInputSchema{
										Slug:         p.GetSlug(),
										ProcessingID: processingId,
									},
									Block: schemas.BlockInputSchema{
										Slug:        _targetBlockSlug,
										Input:       input,
										TargetIndex: _targetBlockInputIndex,
										// Will process only TargetIndex until it reaches DestinationSlug
										// DestinationSlug: _destinationBlockSlug,
									},
								}

								p.Process(
									_workerRegistry,
									_blockRegistry,
									_processingRegistry,
									regenerateData,
									_resultStorages,
								)
							}(
								p,
								workerRegistry,
								blockRegistry,
								processingRegistry,
								resultStorages,
								targetBlockSlug,
								targetBlockInputIndex,
								destinationBlockSlug,
							)
						} else {
							logger.Infof(
								"Pipeline stopped by block [%s:%s]",
								_blockData.GetSlug(),
								block.GetId(),
							)
						}
						return processingOutput, nil
					}

					logger.Infof(
						"Saving output for block [%s:%s] with index %d",
						_blockData.GetSlug(),
						block.GetId(),
						blockInputIndex,
					)

					pipelineBlockDataRegistry.UpdateBlockData(
						_blockData.GetSlug(),
						blockInputIndex,
						processingOutput.GetValue(),
					)

					// Save result to Storage
					saveOutputResults := pipelineBlockDataRegistry.SaveOutput(
						_blockData.GetSlug(),
						blockInputIndex,
						processingOutput.GetValue(),
					)

					for _, saveOutputResult := range saveOutputResults {
						if saveOutputResult.Error != nil {
							logger.Errorf(
								"Error saving output for block [%s:%s] to storage %s: %s",
								_blockData.GetSlug(),
								block.GetId(),
								saveOutputResult.StorageLocation.GetStorageName(),
								saveOutputResult.Error,
							)
						} else {
							logger.Infof(
								"Saved output for block [%s:%s] to storage %s",
								_blockData.GetSlug(),
								block.GetId(),
								saveOutputResult.StorageLocation.GetStorageName(),
							)
						}
					}

					return processingOutput, nil
				}

				if parallel {
					go processBlockInput(
						blockDataClone,
						processing,
						blockInputProcessingResults,
					)
				} else {
					processingOutput, err := processBlockInput(
						blockDataClone,
						processing,
						blockInputProcessingResults,
					)
					if err != nil ||
						processingOutput.GetStop() ||
						processingOutput.GetError() != nil {
						return
					}
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

		logger.Infof("Processing Pipeline %s completed", p.GetSlug())
	}()

	return processingId, nil
}

func (p *PipelineData) GetProcessingsInfo(resultStorages []interfaces.Storage) map[uuid.UUID][]interfaces.PipelineProcessingInfo {
	pipelineProcessingsPath := fmt.Sprintf(
		"%s", p.GetSlug(),
	)

	processingDirectoryRegexp := regexp.MustCompile(
		fmt.Sprintf(
			"%s\\/%s",
			pipelineProcessingsPath,
			"([a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89aAbB][a-f0-9]{3}-[a-f0-9]{12})",
		),
	)

	processings := make(map[uuid.UUID][]interfaces.PipelineProcessingInfo)
	for _, storage := range resultStorages {
		// List all processings in pipelineProcessingsPath
		objects, err := storage.ListObjects(
			storage.NewStorageLocation(
				pipelineProcessingsPath,
			),
		)
		if err != nil || len(objects) == 0 {
			continue
		}

		for _, object := range objects {
			matches := processingDirectoryRegexp.FindStringSubmatch(object.GetFilePath())
			if len(matches) == 2 {
				processingId := uuid.MustParse(matches[1])
				if _, ok := processings[processingId]; !ok {
					processings[processingId] = make(
						[]interfaces.PipelineProcessingInfo,
						0,
					)
				}

				processings[processingId] = append(
					processings[processingId],
					NewPipelineProcessingInfoData(
						processingId,
						object,
					),
				)
			}
		}
	}

	return processings
}

// PipelineProcessingInfoData represents the structure of a pipeline processing in the system.
// It includes the processing's metadata, log, and block data.
//
// swagger:model
type PipelineProcessingInfoData struct {
	sync.Mutex `json:"-"`

	Id          uuid.UUID          `json:"id"`
	Storage     interfaces.Storage `json:"-"`
	IsBlockData bool               `json:"-"`
	BlockData   bytes.Buffer       `json:"-"`
	BlockSlug   string             `json:"-"`
	OutputName  string             `json:"-"`
	IsLogData   bool               `json:"is_log"`
	LogData     string             `json:"log"`
}

func (p *PipelineProcessingInfoData) MarshalJSON() ([]byte, error) {
	customRepresentation := struct {
		Id          uuid.UUID `json:"id"`
		StorageName string    `json:"storage_name"`
		IsLogData   bool      `json:"is_log"`
		LogData     string    `json:"log,omitempty"`
		BlockSlug   string    `json:"block_slug,omitempty"`
		OutputName  string    `json:"output_name,omitempty"`
	}{
		Id:          p.Id,
		StorageName: p.Storage.GetStorageName(),
		IsLogData:   p.IsLogData,
		LogData:     p.LogData,
		BlockSlug:   p.BlockSlug,
		OutputName:  p.OutputName,
	}

	// Use the standard JSON marshaller for the custom structure
	return json.Marshal(customRepresentation)
}

func NewPipelineProcessingInfoData(id uuid.UUID, location interfaces.StorageLocation) *PipelineProcessingInfoData {
	fileName := filepath.Base(location.GetFileName())

	isBlockData := false
	isLogData := false
	logData := ""
	blockslug := ""

	if registries.LOG_FILE_REGEX.MatchString(fileName) {
		isLogData = true
		if logDataBuffer, err := location.GetObjectBytes(); err == nil {
			logData = logDataBuffer.String()
		}
	} else if registries.OUTPUT_FILE_REGEX.MatchString(fileName) {
		isBlockData = true
		segments := strings.Split(location.GetFilePath(), "/")
		for i, segment := range segments {
			if segment == id.String() {
				if i+1 < len(segments) {
					blockslug = segments[i+1]
				}
				break
			}
		}
	}

	return &PipelineProcessingInfoData{
		Id:          id,
		Storage:     location.GetStorage(),
		IsLogData:   isLogData,
		LogData:     logData,
		IsBlockData: isBlockData,
		BlockData:   *bytes.NewBufferString(""),
		BlockSlug:   blockslug,
		OutputName:  fileName,
	}
}

func (p *PipelineProcessingInfoData) GetStorage() interfaces.Storage {
	p.Lock()
	defer p.Unlock()

	return p.Storage
}

func (p *PipelineProcessingInfoData) GetId() uuid.UUID {
	p.Lock()
	defer p.Unlock()

	return p.Id
}

func (p *PipelineProcessingInfoData) SetLogData(data []byte) {
	p.Lock()
	defer p.Unlock()

	p.IsLogData = true
	p.LogData = string(data)
}

func (p *PipelineProcessingInfoData) SetBlockData(data []byte) {
	p.Lock()
	defer p.Unlock()

	p.IsBlockData = true
	p.BlockData.Write(data)
}
