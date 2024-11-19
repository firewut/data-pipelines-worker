package dataclasses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

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
			pipelineBlockDataRegistry.SavePipelineLog(
				loggerBuffer,
				NewPipelineProcessingDetailsFromLogData,
				NewPipelineProcessingStatusFromLogData,
			)
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

				// Get historical data ( previous steps )
				processingData := pipelineBlockDataRegistry.GetAll()

				// If input data passed for the block - remove it from processingData
				if (blockRelativeIndex == 0 &&
					inputData.Block.Slug == blockData.GetSlug()) &&
					inputData.Block.Input != nil &&
					len(inputData.Block.Input) > 0 &&
					inputData.Block.TargetIndex < 0 {

					processingData[inputData.Block.Slug] = nil
				}

				inputConfigValue, parallel, err = blockData.GetInputConfigData(processingData)
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

func (p *PipelineData) GetProcessingsStatus(resultStorages []interfaces.Storage) map[uuid.UUID][]interfaces.PipelineProcessingStatus {
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

	processingsStatuses := make(map[uuid.UUID][]interfaces.PipelineProcessingStatus)
	for _, storage := range resultStorages {
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
				if _, ok := processingsStatuses[processingId]; !ok {
					processingsStatuses[processingId] = make(
						[]interfaces.PipelineProcessingStatus,
						0,
					)
				}

				if registries.STATUS_FILE_REGEX.MatchString(object.GetFilePath()) {
					if statusDataBuffer, err := object.GetObjectBytes(); err == nil {
						processingStatus := NewPipelineProcessingStatusFromStatusFile(
							processingId,
							p.GetSlug(),
							statusDataBuffer,
							storage,
						)

						processingsStatuses[processingId] = append(
							processingsStatuses[processingId],
							processingStatus,
						)
					}
				}
			}
		}
	}

	return processingsStatuses
}

func (p *PipelineData) GetProcessingDetails(processingId uuid.UUID, resultStorages []interfaces.Storage) []interfaces.PipelineProcessingDetails {
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

	processings := make([]interfaces.PipelineProcessingDetails, 0)
	for _, storage := range resultStorages {
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
				storageProcessingId := uuid.MustParse(matches[1])
				if storageProcessingId != processingId {
					continue
				}

				if registries.LOG_FILE_REGEX.MatchString(object.GetFilePath()) {
					if statusDataBuffer, err := object.GetObjectBytes(); err == nil {
						processingDetails := NewProcessingDetailsFromLogFile(
							processingId,
							p.GetSlug(),
							statusDataBuffer,
							storage,
						)

						processings = append(processings, processingDetails)
					}
				}
			}
		}
	}

	return processings
}

// PipelineProcessingStatus represents the structure of a pipeline processing status in the system.
// It includes the processing's metadata, storage, and completion status.
//
// swagger:model
type PipelineProcessingStatus struct {
	Id           uuid.UUID `json:"id"`
	PipelineSlug string    `json:"pipeline_slug"`
	LogId        uuid.UUID `json:"log_id"`
	Storage      string    `json:"storage"`
	IsStopped    bool      `json:"is_stopped"`
	IsCompleted  bool      `json:"is_completed"`
	IsError      bool      `json:"is_error"`
	DateFinished time.Time `json:"date_finished"`
}

func (p *PipelineProcessingStatus) GetId() uuid.UUID {
	return p.Id
}

func (p *PipelineProcessingStatus) MarshalJSON() ([]byte, error) {
	customRepresentation := struct {
		Id           uuid.UUID `json:"id"`
		LogId        uuid.UUID `json:"log_id"`
		Storage      string    `json:"storage"`
		IsStopped    bool      `json:"is_stopped"`
		IsCompleted  bool      `json:"is_completed"`
		IsError      bool      `json:"is_error"`
		DateFinished time.Time `json:"date_finished"`
	}{
		Id:           p.Id,
		LogId:        p.LogId,
		Storage:      p.Storage,
		IsStopped:    p.IsStopped,
		IsCompleted:  p.IsCompleted,
		IsError:      p.IsError,
		DateFinished: p.DateFinished,
	}

	return json.Marshal(customRepresentation)
}

func NewPipelineProcessingStatusFromStatusFile(
	id uuid.UUID,
	pipelineSlug string,
	statusFileBuffer *bytes.Buffer,
	storage interfaces.Storage,
) interfaces.PipelineProcessingStatus {
	processingStatus := &PipelineProcessingStatus{
		Id:           id,
		PipelineSlug: pipelineSlug,
		Storage:      storage.GetStorageName(),
	}

	if err := json.Unmarshal(statusFileBuffer.Bytes(), processingStatus); err != nil {
		config.GetLogger().Error(err)
	}

	return processingStatus
}

func NewPipelineProcessingStatusFromLogData(
	id uuid.UUID,
	pipelineSlug string,
	logId uuid.UUID,
	logBuffer *bytes.Buffer,
	storage interfaces.Storage,
) interfaces.PipelineProcessingStatus {
	is_stopped := false
	is_completed := false
	is_error := false

	logData := logBuffer.String()

	// Check that pipeline is correct
	if strings.Contains(
		logData, fmt.Sprintf(`"prefix":"pipeline:%s"`, id.String()),
	) {
		if strings.Contains(
			logData, fmt.Sprintf(`"message":"Processing Pipeline %s completed"`, pipelineSlug),
		) {
			is_completed = true
		}
		if strings.Contains(
			logData, `"message":"Pipeline stopped by block [`,
		) {
			is_stopped = true
		}
	}

	if strings.Contains(
		logData,
		`"ERROR"`,
	) {
		is_error = true
	}

	return &PipelineProcessingStatus{
		Id:           id,
		Storage:      storage.GetStorageName(),
		PipelineSlug: pipelineSlug,
		LogId:        logId,
		IsStopped:    is_stopped,
		IsCompleted:  is_completed,
		IsError:      is_error,
		DateFinished: time.Now().UTC(),
	}
}

// PipelineProcessingDetails represents the structure of a pipeline processing in the system.
// It includes the processing's metadata, log, and block data.
//
// swagger:model
type PipelineProcessingDetails struct {
	PipelineProcessingStatus
	LogData []map[string]interface{} `json:"log_data"`
}

func (p *PipelineProcessingDetails) GetId() uuid.UUID {
	return p.Id
}

func (p *PipelineProcessingDetails) MarshalJSON() ([]byte, error) {
	customRepresentation := struct {
		Id           uuid.UUID                `json:"id"`
		PipelineSlug string                   `json:"pipeline_slug"`
		LogId        uuid.UUID                `json:"log_id"`
		Storage      string                   `json:"storage"`
		IsStopped    bool                     `json:"is_stopped"`
		IsCompleted  bool                     `json:"is_completed"`
		IsError      bool                     `json:"is_error"`
		DateFinished time.Time                `json:"date_finished"`
		LogData      []map[string]interface{} `json:"log_data"`
	}{
		Id:           p.Id,
		PipelineSlug: p.PipelineSlug,
		LogId:        p.LogId,
		Storage:      p.Storage,
		IsStopped:    p.IsStopped,
		IsCompleted:  p.IsCompleted,
		IsError:      p.IsError,
		DateFinished: p.DateFinished,
		LogData:      p.LogData,
	}

	return json.Marshal(customRepresentation)
}

func NewProcessingDetailsFromLogFile(
	id uuid.UUID,
	slug string,
	logBuffer *bytes.Buffer,
	storage interfaces.Storage,
) interfaces.PipelineProcessingDetails {
	processingDetails := &PipelineProcessingDetails{
		PipelineProcessingStatus: PipelineProcessingStatus{
			Id:           id,
			PipelineSlug: slug,
			Storage:      storage.GetStorageName(),
		},
	}

	if err := json.Unmarshal(logBuffer.Bytes(), processingDetails); err != nil {
		config.GetLogger().Error(err)
	}

	return processingDetails
}

func NewPipelineProcessingDetailsFromLogData(
	id uuid.UUID,
	slug string,
	logId uuid.UUID,
	logBuffer *bytes.Buffer,
	storage interfaces.Storage,
) interfaces.PipelineProcessingDetails {
	status := NewPipelineProcessingStatusFromLogData(id, slug, logId, logBuffer, storage)

	logData := make([]map[string]interface{}, 0)
	for _, logLine := range strings.Split(logBuffer.String(), "\n") {
		logDataItem := make(map[string]interface{})
		if err := json.Unmarshal([]byte(logLine), &logDataItem); err == nil {
			logData = append(logData, logDataItem)
		}
	}

	return &PipelineProcessingDetails{
		PipelineProcessingStatus: *status.(*PipelineProcessingStatus),
		LogData:                  logData,
	}
}
